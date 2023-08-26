package cluster

// import (
// 	"log/slog"
// 	"path"
// 	"runtime"
// 	"time"

// 	"github.com/libp2p/go-libp2p"
// 	local "github.com/libp2p/go-libp2p/core/host"
// 	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
// 	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
// 	"github.com/urfave/cli/v2"

// 	"github.com/wetware/pkg/cap/host"
// 	"github.com/wetware/pkg/client"
// 	"github.com/wetware/pkg/system"
// )

// var (
// 	h        host.Host
// 	releases *[]func()
// 	closes   *[]func() error
// )

// var flags = []cli.Flag{
// 	&cli.StringSliceFlag{
// 		Name:    "addr",
// 		Aliases: []string{"a"},
// 		Usage:   "static bootstrap `ADDR`",
// 		EnvVars: []string{"WW_ADDR"},
// 	},
// 	&cli.StringFlag{
// 		Name:    "discover",
// 		Aliases: []string{"d"},
// 		Usage:   "bootstrap discovery `ADDR`",
// 		Value:   bootstrapAddr(),
// 		EnvVars: []string{"WW_DISCOVER"},
// 	},
// 	&cli.StringFlag{
// 		Name:    "ns",
// 		Usage:   "cluster namespace",
// 		Value:   "ww",
// 		EnvVars: []string{"WW_NS"},
// 	},
// 	&cli.DurationFlag{
// 		Name:    "timeout",
// 		Usage:   "dial timeout",
// 		Value:   time.Second * 15,
// 		EnvVars: []string{"WW_CLIENT_TIMEOUT"},
// 	},
// }

// func Command() *cli.Command {
// 	return &cli.Command{
// 		Name:    "cluster",
// 		Usage:   "cli client for wetware clusters",
// 		Aliases: []string{"client"}, // TODO(soon):  deprecate
// 		Flags:   flags,
// 		Subcommands: []*cli.Command{
// 			run(),
// 		},
// 	}
// }

// func setup() cli.BeforeFunc {
// 	return func(c *cli.Context) (err error) {
// 		*releases = make([]func(), 0)
// 		*closes = make([]func() error, 0)

// 		h, err := clientHost(c)
// 		if err != nil {
// 			return err
// 		}
// 		*closes = append(*closes, h.Close)

// 		host, err := system.Bootstrap[host.Host](c.Context, h, client.Config{
// 			Logger:   slog.Default(),
// 			NS:       c.String("ns"),
// 			Peers:    c.StringSlice("peer"),
// 			Discover: c.String("discover"),
// 		})
// 		if err != nil {
// 			return err
// 		}
// 		*releases = append(*releases, host.Release)

// 		return nil
// 	}
// }

// func teardown() cli.AfterFunc {
// 	return func(c *cli.Context) (err error) {
// 		for _, close := range *closes {
// 			defer close()
// 		}
// 		for _, release := range *releases {
// 			defer release()
// 		}
// 		return nil
// 	}
// }

// func clientHost(c *cli.Context) (local.Host, error) {
// 	return libp2p.New(
// 		libp2p.NoTransports,
// 		libp2p.NoListenAddrs,
// 		libp2p.Transport(tcp.NewTCPTransport),
// 		libp2p.Transport(quic.NewTransport))
// }

// func bootstrapAddr() string {
// 	return path.Join("/ip4/228.8.8.8/udp/8822/multicast", loopback())
// }

// func loopback() string {
// 	switch runtime.GOOS {
// 	case "darwin":
// 		return "lo0"
// 	default:
// 		return "lo"
// 	}
// }
