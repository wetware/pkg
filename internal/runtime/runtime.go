package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"
	"github.com/thejerf/suture/v4"
	"github.com/urfave/cli/v2"
	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/casm/pkg/pex"
	serviceutil "github.com/wetware/ww/internal/util/service"
	statsdutil "github.com/wetware/ww/internal/util/statsd"
	"github.com/wetware/ww/pkg/server"
	"github.com/wetware/ww/pkg/vat"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"
)

/****************************************************************************
 *                                                                          *
 *  runtime.go is responsible for managing the lifetimes of services.       *
 *                                                                          *
 ****************************************************************************/

var localnode = fx.Provide(
	supervisor,
	node)

func Serve(c *cli.Context) error {
	var app = fx.New(fx.NopLogger,
		fx.Supply(c),
		system,
		network,
		localnode,
		fx.Invoke(bind))

	if err := start(c, app); err != nil {
		return err
	}

	<-app.Done() // TODO:  process OS signals in a loop here.

	return shutdown(app)
}

func start(c *cli.Context, app *fx.App) error {
	ctx, cancel := context.WithTimeout(c.Context, time.Second*15)
	defer cancel()

	return app.Start(ctx)
}

func shutdown(app *fx.App) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	if err = app.Stop(ctx); err == context.Canceled {
		err = nil
	}

	return
}

// Config declares dependencies that are dynamically resolved at
// runtime.
type Config struct {
	fx.In

	Lifecycle fx.Lifecycle

	Logger     log.Logger
	Node       *server.Node
	Supervisor *suture.Supervisor
	Services   []suture.Service `group:"services"` // caller-supplied services
}

func (config Config) Log() log.Logger {
	return config.Logger.With(config.Node)
}

func bind(c *cli.Context, config Config) {
	ctx, cancel := context.WithCancel(c.Context) // cancelled by stop hook

	// Bind user-defined services to the runtime.
	for _, service := range config.Services {
		config.Supervisor.Add(service)
	}

	// Set up some shared variables.  These will mutate throughout the application
	// lifecycle.
	var cherr <-chan error

	// Hook main service (Supervisor) into application lifecycle.
	config.Lifecycle.Append(fx.Hook{
		// Bind global variables and start wetware.
		OnStart: func(_ context.Context) error {
			cherr = config.Supervisor.ServeBackground(ctx) // NOTE: application context

			// The wetware environment is now loaded.  Bear in mind
			// that this is a PA/EL system, so we hand over control
			// to the user without synchronizing the cluster view.
			//
			// System events are likewise asynchronous, but reliable.
			// Users can wait for the local node to have successfully
			// bound to network interfaces by subscribing to the
			// 'EvtLocalAddrsUpdated' event.

			config.Log().Info("wetware loaded")

			return nil
		},
		OnStop: func(ctx context.Context) (err error) {
			// Cancel the application context.
			//
			// Beyond this point, the services in this local
			// process are no longer guaranteed to be in sync.
			cancel()

			// Wait for the supervisor to shut down gracefully.
			// If it does not terminate in a timely fashion,
			// abort the application and return an error.
			select {
			case err = <-cherr:
				return err

			case <-ctx.Done():
				return fmt.Errorf("shutdown: %w", ctx.Err())
			}
		},
	})
}

//
// Dependency declarations
//

func supervisor(c *cli.Context) *suture.Supervisor {
	return suture.New("runtime", suture.Spec{
		EventHook: serviceutil.NewEventHook(c),
	})
}

type serverConfig struct {
	fx.In

	Log     log.Logger
	Vat     vat.Network
	PubSub  *pubsub.PubSub
	PeX     *pex.PeerExchange
	DHT     *dual.DHT
	Metrics *statsdutil.WwMetricsRecorder

	Lifecycle fx.Lifecycle
}

func (config serverConfig) Logger() log.Logger {
	return config.Log.With(config.Vat)
}

func (config serverConfig) ClusterOpts() []cluster.Option {
	return []cluster.Option{
		cluster.WithMeta(nil)}
}

func (config serverConfig) SetCloser(c io.Closer) {
	config.Lifecycle.Append(closer(c))
}

func node(c *cli.Context, config serverConfig) (*server.Node, error) {
	n, err := server.New(c.Context, config.Vat, config.PubSub,
		server.WithLogger(config.Logger()),
		server.WithClusterConfig(config.ClusterOpts()...),
		server.WithMetrics(config.Metrics))

	if err == nil {
		config.SetCloser(n)
	}

	return n, err
}

type mergeFromPeX struct {
	ns  string
	pex *pex.PeerExchange
	dht *dual.DHT
}

func (m mergeFromPeX) Merge(ctx context.Context, peers []peer.AddrInfo) error {
	var g errgroup.Group

	for _, info := range peers {
		g.Go(m.merger(ctx, info))
	}

	return g.Wait()

}

func (m mergeFromPeX) merger(ctx context.Context, info peer.AddrInfo) func() error {
	return func() error {
		if err := m.PerformGossipRound(ctx, info); err != nil {
			return fmt.Errorf("%s: %w", info.ID.ShortString(), err)
		}

		return m.RefreshDHT(ctx)
	}
}

func (m mergeFromPeX) PerformGossipRound(ctx context.Context, info peer.AddrInfo) (err error) {
	if err = m.pex.Bootstrap(ctx, m.ns, info); err != nil {
		err = fmt.Errorf("pex: %w", err)
	}

	return
}

func (m mergeFromPeX) RefreshDHT(ctx context.Context) error {
	// FIXME:  implement DHT refresh
	return errors.New("NOT IMPLEMENTED")
}

func closer(c io.Closer) fx.Hook {
	return fx.Hook{
		OnStop: onclose(c),
	}
}

func onclose(c io.Closer) func(context.Context) error {
	return func(context.Context) error {
		return c.Close()
	}
}
