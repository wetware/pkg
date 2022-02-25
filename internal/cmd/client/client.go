package client

import (
	"github.com/lthibault/log"
	"github.com/urfave/cli/v2"
)

var (
	logger log.Logger
	// node   *client.Node
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "client",
		Usage: "cli client for wetware clusters",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "discover",
				Aliases: []string{"d"},
				Usage:   "bootstrap discovery addr in URL-CIDR format",
				Value:   "tcp://127.0.0.1:8822/24", // TODO:  this should default to mudp
			},
			&cli.StringFlag{
				Name:    "ns",
				Usage:   "cluster namespace",
				Value:   "ww",
				EnvVars: []string{"WW_NS"},
			},
		},
		Subcommands: commands,
	}
}

var commands = []*cli.Command{
	Discover(),
	// Publish(),
	Subscribe(),
}

// ww client discover
func Discover() *cli.Command {
	return &cli.Command{
		Name:  "discover",
		Usage: "bootstrap client",
		Subcommands: []*cli.Command{
			Crawl(),
			Publish(),
		},
	}
}

// ww client subscribe <topic>
func Subscribe() *cli.Command {
	return &cli.Command{
		Name:    "subscribe",
		Aliases: []string{"sub"},
		Usage:   "subscribe to a pubsub topic",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "topic",
				Usage: "pubsub topic (\"\" is the cluster topic)",
			},
		},
		// Before: before,
		Action: subscribe,
	}
}

// func before(c *cli.Context) error {
// 	boot, err := bootutil.NewCrawler(c)
// 	if err != nil {
// 		return err
// 	}

// 	h, err := libp2p.New(c.Context,
// 		libp2p.NoTransports,
// 		libp2p.NoListenAddrs,
// 		libp2p.Transport(libp2pquic.NewTransport))
// 	if err != nil {
// 		return err
// 	}

// 	node, err = client.Dialer{
// 		Boot: boot,
// 		Vat: vat.Network{
// 			NS:   c.String("ns"),
// 			Host: h,
// 		},
// 	}.Dial(c.Context)

// 	return err
// }
