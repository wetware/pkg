package runtime

import (
	"context"
	"fmt"
	"io"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"
	"github.com/thejerf/suture/v4"
	"github.com/urfave/cli/v2"
	"github.com/wetware/casm/pkg/cluster"
	logutil "github.com/wetware/ww/internal/util/log"
	serviceutil "github.com/wetware/ww/internal/util/service"
	statsdutil "github.com/wetware/ww/internal/util/statsd"
	"github.com/wetware/ww/pkg/server"
	"github.com/wetware/ww/pkg/vat"
	"go.uber.org/fx"
)

/****************************************************************************
 *                                                                          *
 *  runtime.go is responsible for managing the lifetimes of services.       *
 *                                                                          *
 ****************************************************************************/

var (
	instrumentation = fx.Provide(
		statsdutil.NewBandwidthCounter,
		statsdutil.New,
		logutil.New)

	localnode = fx.Provide(
		supervisor,
		node)
)

func Serve(c *cli.Context) error {
	var app = fx.New(fx.NopLogger,
		fx.Supply(c),
		instrumentation,
		localnode,
		network,
		system,
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

	Log       log.Logger
	Vat       vat.Network
	PubSub    *pubsub.PubSub
	Lifecycle fx.Lifecycle
}

func node(c *cli.Context, config serverConfig) (*server.Node, error) {
	n, err := server.New(c.Context, config.Vat, config.PubSub,
		server.WithLogger(config.Log),
		server.WithClusterConfig(
			cluster.WithMeta(nil) /* TODO */))

	if err == nil {
		config.Lifecycle.Append(closer(n))
	}

	return n, err
}

func closer(c io.Closer) fx.Hook {
	return fx.Hook{
		OnStop: func(context.Context) error {
			return c.Close()
		},
	}
}
