package cluster

import (
	"context"
	"log"
	"path"
	"runtime"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/client"
	"go.uber.org/fx"
)

var (
	app    *fx.App
	h      host.Host // TODO, @lthibault could you lend me a hand? Let's talk in matrix :)
	logger log.Logger
	dialer client.Dialer
)

var subcommands = []*cli.Command{
	run(),
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
		Value:   bootstrapAddr(),
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
		ctx, cancel := context.WithTimeout(
			c.Context,
			c.Duration("timeout"))
		defer cancel()

		// TODO: populate h

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
