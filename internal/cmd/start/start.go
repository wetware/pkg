package start

import (
	"github.com/urfave/cli/v2"
	"github.com/wetware/ww/internal/runtime"
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
		Aliases: []string{"a"},
		Usage:   "host listen address",
		Value: cli.NewStringSlice(
			"/ip4/0.0.0.0/udp/0/quic",
			"/ip6/::0/udp/0/quic"),
		EnvVars: []string{"WW_LISTEN"},
	},
	&cli.StringSliceFlag{
		Name:    "join",
		Usage:   "join cluster through current member address",
		EnvVars: []string{"WW_JOIN"},
	},
	&cli.StringFlag{
		Name:    "discover",
		Aliases: []string{"d"},
		Usage:   "bootstrap discovery multiaddr",
		Value:   "/ip4/228.8.8.8/udp/8822/multicast/lo0",
		EnvVars: []string{"WW_DISCOVER"},
	},
}

// Command constructor
func Command() *cli.Command {
	return &cli.Command{
		Name:   "start",
		Usage:  "start a host process",
		Flags:  flags,
		Action: runtime.Serve,
	}
}
