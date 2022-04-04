package client

import (
	"github.com/lthibault/log"
	"github.com/urfave/cli/v2"
	logutil "github.com/wetware/ww/internal/util/log"
)

var logger log.Logger

var subcommands = []*cli.Command{
	Ls(),
	Join(),
}

func Command() *cli.Command {
	return &cli.Command{
		Name:  "client",
		Usage: "cli client for wetware clusters",
		Flags: []cli.Flag{
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
		},
		Subcommands: subcommands,

		Before: func(c *cli.Context) error {
			logger = logutil.New(c)
			return nil
		},

		// NOTE:  Do not call dial() here because certain commands may not
		//        require a client node.  The shutdown hook checks whether
		//        a client node was instantiated before calling Close().
		After: shutdown(),
	}
}
