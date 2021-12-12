package start

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/lthibault/log"
	"github.com/thejerf/suture/v4"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"

	serviceutil "github.com/wetware/ww/internal/util/service"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/server"
	"github.com/wetware/ww/pkg/util/embed"
)

var logger = log.New()

var flags = []cli.Flag{
	&cli.StringFlag{
		Name:    "join",
		Aliases: []string{"j"},
		Usage:   "addrs to static bootstrap peers",
		EnvVars: []string{"WW_JOIN"},
	},
	&cli.StringFlag{
		Name:    "discover",
		Aliases: []string{"d"},
		Usage:   "bootstrap service multiaddr",
		Value:   "/ip4/228.8.8.8/udp/8822",
		EnvVars: []string{"WW_DISCOVER"},
	},
	&cli.StringSliceFlag{
		Name:    "addr",
		Aliases: []string{"a"},
		Usage:   "host listen address",
		Value: cli.NewStringSlice(
			"/ip4/0.0.0.0/udp/2020/quic",
			"/ip6/::0/udp/2020/quic"),
		EnvVars: []string{"WW_ADDR"},
	},
	&cli.StringFlag{
		Name:    "ns",
		Usage:   "cluster namespace",
		Value:   "ww",
		EnvVars: []string{"WW_NS"},
	},
	&cli.DurationFlag{
		Name:    "ttl",
		Usage:   "heartbeat TTL duration",
		Value:   time.Second * 5,
		EnvVars: []string{"WW_TTL"},
	},
	&cli.StringFlag{
		Name:    "secret",
		Usage:   "cluster-wide shared secret",
		EnvVars: []string{"WW_SECRET"},
	},
}

// SetLogger assigns the global logger for this command module.
// It has no effect after the Command().Action has begune executing.
func SetLogger(log log.Logger) { logger = log }

// Command constructor
func Command() *cli.Command {
	return &cli.Command{
		Name:   "start",
		Usage:  "start a host process",
		Flags:  flags,
		Action: run(),
	}
}

func run() cli.ActionFunc {
	return func(c *cli.Context) error {
		app := fx.New(fx.NopLogger,
			newSharedSecret(c),
			fx.Supply(c,
				fx.Annotate(c.String("ns"), fx.ParamTags(`name:"ns"`)),
				fx.Annotate(c.Duration("ttl"), fx.ParamTags(`name:"ttl"`)),
				fx.Annotate(c.Context, fx.As(new(context.Context))),
				fx.Annotate(logger, fx.As(new(log.Logger)))),
			fx.Provide(
				newBootStrategy,
				newSystemHook,
				newDatastore,
				embed.Server),
			fx.Invoke(
				startRuntime))

		if err := app.Start(c.Context); err != nil {
			return err
		}

		<-c.Context.Done()

		return shutdown(app)
	}
}

type runtime struct {
	fx.In

	Lifecycle fx.Lifecycle

	Logger log.Logger
	CLI    *cli.Context

	Bootstrap discovery.Advertiser
	Services  []suture.Service `group:"services"`
}

func startRuntime(ctx context.Context, n server.Node, r runtime) {
	// Set up some shared variables.  These will mutate throughout the application
	// lifecycle.
	var (
		cancel context.CancelFunc
		cherr  <-chan error
	)

	r.Logger = r.Logger.WithField("version", ww.Version)

	// Register services.
	s := serviceutil.New(r.Logger.With(n), r.CLI.String("ns"))
	s.Add(r)

	// Hook main service (Supervisor) into application lifecycle.
	r.Lifecycle.Append(fx.Hook{
		// Bind global variables and start wetware.
		OnStart: func(context.Context) error {
			ctx, cancel = context.WithCancel(ctx)

			// Start the supervisor.
			cherr = s.ServeBackground(ctx)

			// The wetware environment is now loaded.  Bear in mind
			// that this is a PA/EL system, so we hand over control
			// to the user without synchronizing the cluster view.
			//
			// System events are likewise asynchronous, but reliable.
			// Users can wait for the local node to have successfully
			// bound to network interfaces by subscribing to the
			// 'EvtLocalAddrsUpdated' event.
			r.Logger.WithField("ns", s).Debug("loaded namespace")

			return nil
		},
		OnStop: func(ctx context.Context) (err error) {
			// Cancel the application context.
			//
			// Beyond this point, the services in this local
			// process are no longer guaranteed to be in sync.
			cancel()
			r.Logger.Tracef("shutdown signal sent to %s", s)

			// Wait for the supervisor to shut down gracefully.
			// If it does not terminate in a timely fashion,
			// abort the application and return an error.
			select {
			case err = <-cherr:
				if err == context.Canceled {
					err = nil
				}

			case <-ctx.Done():
				err = ctx.Err()
			}

			return
		},
	})
}

func (r runtime) Serve(ctx context.Context) error {
	r.Logger.Info("wetware started")

	g, ctx := errgroup.WithContext(ctx)
	for _, s := range r.Services {
		g.Go(serve(ctx, s))
	}

	// Advertise our presence to the network.
	_, err := r.Bootstrap.Advertise(ctx, r.CLI.String("ns"))
	if err != nil {
		return err
	}

	return g.Wait()
}

func serve(ctx context.Context, s suture.Service) func() error {
	return func() error {
		return s.Serve(ctx)
	}
}

func shutdown(app *fx.App) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	return app.Stop(ctx)
}
