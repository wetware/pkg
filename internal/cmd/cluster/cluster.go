package cluster

import (
	"context"
	"time"

	"github.com/lthibault/log"
	"github.com/urfave/cli/v2"
	runtimeutil "github.com/wetware/ww/internal/util/runtime"
	"github.com/wetware/ww/pkg/client"
	"github.com/wetware/ww/pkg/runtime"
	"go.uber.org/fx"
)

var (
	app    *fx.App
	node   *client.Node
	logger log.Logger
	dialer client.Dialer
)

var subcommands = []*cli.Command{
	list(),
	publish(),
	subscribe(),
	// Discover(),
}

var flags = []cli.Flag{
	&cli.StringSliceFlag{
		Name:    "addr",
		Aliases: []string{"a"},
		Usage:   "static bootstrap `ADDR`",
		EnvVars: []string{"WW_ADDR"},
	},
	&cli.StringFlag{
		Name:    "discover",
		Aliases: []string{"d"},
		Usage:   "bootstrap discovery `ADDR`",
		Value:   "/ip4/228.8.8.8/udp/8822/multicast/lo0",
		EnvVars: []string{"WW_DISCOVER"},
	},
	&cli.StringFlag{
		Name:    "ns",
		Usage:   "cluster namespace",
		Value:   "ww",
		EnvVars: []string{"WW_NS"},
	},
	&cli.DurationFlag{
		Name:    "timeout",
		Usage:   "dial timeout",
		Value:   time.Second * 15,
		EnvVars: []string{"WW_CLIENT_TIMEOUT"},
	},
}

func Command() *cli.Command {
	return &cli.Command{
		Name:        "cluster",
		Usage:       "cli client for wetware clusters",
		Aliases:     []string{"client"}, // TODO(soon):  deprecate
		Flags:       flags,
		Subcommands: subcommands,
	}
}

func setup() cli.BeforeFunc {
	return func(c *cli.Context) (err error) {
		app = fx.New(
			runtime.Prelude(runtimeutil.New(c)),
			fx.StartTimeout(c.Duration("timeout")),
			fx.Populate(&logger, &dialer),
			runtime.Client())

		ctx, cancel := context.WithTimeout(
			c.Context,
			c.Duration("timeout"))
		defer cancel()

		if node, err = dialer.Dial(ctx); err != nil {
			return
		}

		return app.Start(ctx)
	}
}

func teardown() cli.AfterFunc {
	return func(c *cli.Context) (err error) {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			app.StopTimeout())
		defer cancel()
		return app.Stop(ctx)
	}
}
