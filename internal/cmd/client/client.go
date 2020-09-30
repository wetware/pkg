package client

import (
	"context"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/wetware/ww/pkg/client"

	clientutil "github.com/wetware/ww/internal/util/client"
	ctxutil "github.com/wetware/ww/internal/util/ctx"
)

var (
	root client.Client // see before()
	ctx  = ctxutil.WithDefaultSignals(context.Background())

	flags = []cli.Flag{
		&cli.StringSliceFlag{
			Name:    "join",
			Aliases: []string{"j"},
			Usage:   "connect to cluster through specified peers",
			EnvVars: []string{"WW_JOIN"},
		},
		&cli.StringFlag{
			Name:    "discover",
			Aliases: []string{"d"},
			Usage:   "automatic peer discovery settings",
			Value:   "/mdns",
			EnvVars: []string{"WW_DISCOVER"},
		},
		&cli.StringFlag{
			Name:    "namespace",
			Aliases: []string{"ns"},
			Usage:   "cluster namespace (must match dial host)",
			Value:   "ww",
			EnvVars: []string{"WW_NAMESPACE"},
		},
		&cli.DurationFlag{
			Name:  "timeout",
			Usage: "timeout for -dial",
			Value: time.Second * 10,
		},
	}
)

// Command constructor
func Command() *cli.Command {
	return &cli.Command{
		Name:        "client",
		Usage:       "interact with a live cluster",
		Flags:       flags,
		Before:      before(),
		After:       after(),
		Subcommands: subcommands(),
	}
}

// before the wetware client
func before() cli.BeforeFunc {
	return func(c *cli.Context) (err error) {
		ctx, cancel := context.WithTimeout(ctx, c.Duration("timeout"))
		defer cancel()

		root, err = clientutil.Dial(ctx, c)
		return
	}
}

func after() cli.AfterFunc {
	return func(c *cli.Context) error {
		return root.Close()
	}
}

func subcommands() []*cli.Command {
	return []*cli.Command{
		ls(),
		subscribe(),
		publish(),
	}
}
