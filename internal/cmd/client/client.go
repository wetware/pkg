package client

import (
	"time"

	"github.com/urfave/cli/v2"
)

var subcommands = []*cli.Command{
	Ls(),
	Join(),
	Publish(),
	Subscribe(),
	Discover(),
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
			&cli.DurationFlag{
				Name:    "timeout",
				Usage:   "dial timeout",
				Value:   time.Second * 15,
				EnvVars: []string{"WW_CLIENT_TIMEOUT"},
			},
		},
		Subcommands: subcommands,

		Before: setup(),
		After:  teardown(),
	}
}
