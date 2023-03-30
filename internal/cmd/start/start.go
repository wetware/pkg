package start

import (
	"context"
	"path"
	"runtime"

	"github.com/lthibault/log"

	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	ww_runtime "github.com/wetware/ww/pkg/runtime"
	"github.com/wetware/ww/pkg/server"
)

var (
	app    *fx.App
	logger log.Logger
	node   *server.Node
)

var flags = []cli.Flag{
	&cli.StringFlag{
		Name:    "ns",
		Usage:   "cluster namespace",
		Value:   "ww",
		EnvVars: []string{"WW_NS"},
	},
	&cli.StringSliceFlag{
		Name:    "listen",
		Aliases: []string{"l"},
		Usage:   "host listen address",
		Value: cli.NewStringSlice(
			"/ip4/0.0.0.0/udp/0/quic",
			"/ip6/::0/udp/0/quic"),
		EnvVars: []string{"WW_LISTEN"},
	},
	&cli.StringSliceFlag{
		Name:    "addr",
		Aliases: []string{"a"},
		Usage:   "static bootstrap `ADDR`",
		EnvVars: []string{"WW_ADDR"},
	},
	&cli.StringFlag{
		Name:    "discover",
		Aliases: []string{"d"},
		Usage:   "bootstrap discovery multiaddr",
		Value:   bootstrapAddr(),
		EnvVars: []string{"WW_DISCOVER"},
	},
	&cli.StringSliceFlag{
		Name:    "meta",
		Usage:   "metadata fields in key=value format",
		EnvVars: []string{"WW_META"},
	},
}

// Command constructor
func Command() *cli.Command {
	return &cli.Command{
		Name:   "start",
		Usage:  "start a host process",
		Flags:  flags,
		Before: setup(),
		Action: serve(),
		After:  teardown(),
	}
}

func setup() cli.BeforeFunc {
	return func(c *cli.Context) error {
		app = fx.New(
			ww_runtime.NewServer(c.Context, c),
			fx.Populate(&logger, &node))

		return start(c.Context, app)
	}
}

func serve() cli.ActionFunc {
	return func(*cli.Context) error {
		logger.Info("wetware started")

		signal := <-app.Done()
		logger.
			WithField("signal", signal).
			Warn("shutdown signal received")

		return nil
	}
}

func start(ctx context.Context, app *fx.App) error {
	ctx, cancel := context.WithTimeout(ctx, app.StartTimeout())
	defer cancel()

	return app.Start(ctx)
}

func teardown() cli.AfterFunc {
	return func(c *cli.Context) error {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			app.StopTimeout())
		defer cancel()

		return app.Stop(ctx)
	}
}

func bootstrapAddr() string {
	return path.Join("/ip4/228.8.8.8/udp/8822/multicast", loopback())
}

func loopback() string {
	switch runtime.GOOS {
	case "darwin":
		return "lo0"
	default:
		return "lo"
	}
}
