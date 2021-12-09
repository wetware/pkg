package start

import (
	"context"
	"time"

	"github.com/lthibault/log"
	"github.com/thejerf/suture/v4"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	logutil "github.com/wetware/ww/internal/util/log"
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
				newSystemHook,
				newDatastore),
			fx.Invoke(start))

		if err := app.Start(c.Context); err != nil {
			return err
		}

		<-c.Context.Done()

		return shutdown(app)
	}
}

func start(ctx context.Context, cfg embed.ServerConfig, lx fx.Lifecycle) {
	var (
		n = embed.Server(cfg)
		s = suture.New("ww", suture.Spec{
			EventHook: logutil.NewEventHook(logger),
		})

		cancel context.CancelFunc
		cherr  <-chan error
	)

	s.Add(n)

	lx.Append(fx.Hook{
		OnStart: func(context.Context) error {
			ctx, cancel = context.WithCancel(ctx)
			cherr = s.ServeBackground(ctx)
			return nil
		},
		OnStop: func(ctx context.Context) (err error) {
			cancel()

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

func shutdown(app *fx.App) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	return app.Stop(ctx)
}
