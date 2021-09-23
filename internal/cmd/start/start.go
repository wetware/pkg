package start

import (
	"context"
	"io"
	"time"

	"github.com/lthibault/log"
	"github.com/urfave/cli/v2"
	ww "github.com/wetware/ww/pkg"
	"go.uber.org/fx"
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
			fx.Supply(c,
				fx.Annotate(c.String("ns"), fx.ParamTags(`name:"ns"`)),
				fx.Annotate(c.Context, fx.As(new(context.Context))),
				fx.Annotate(logger, fx.As(new(log.Logger)))),
			fx.Provide(
				newSystemHook,
				newDatastore,
				newRoutedHost,
				newPubSub),
			fx.Invoke(start))

		if err := app.Start(c.Context); err != nil {
			return err
		}

		<-c.Context.Done()

		return shutdown(app)
	}
}

func start(cfg ww.Config, lx fx.Lifecycle) {
	var n ww.Node
	lx.Append(fx.Hook{
		OnStart: func(ctx context.Context) (err error) {
			logger.Info("ready")
			n, err = ww.New(ctx, cfg)
			return
		},
		OnStop: func(ctx context.Context) error {
			logger.Warn("shutting down")
			return n.Shutdown(ctx)
		},
	})
}

func shutdown(app *fx.App) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	return app.Stop(ctx)
}

func closer(c io.Closer) fx.Hook {
	return fx.Hook{
		OnStop: func(context.Context) error {
			return c.Close()
		},
	}
}
