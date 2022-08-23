package client

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"time"

// 	"github.com/urfave/cli/v2"

// 	"github.com/libp2p/go-libp2p"
// 	"github.com/libp2p/go-libp2p/core/discovery"
// 	"github.com/libp2p/go-libp2p/core/peer"
// 	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
// 	"github.com/wetware/casm/pkg/boot/socket"
// 	bootutil "github.com/wetware/casm/pkg/boot/util"
// 	logutil "github.com/wetware/ww/internal/util/log"
// )

// func Discover() *cli.Command {
// 	return &cli.Command{
// 		Name:  "discover",
// 		Usage: "discover a wetware node and print its multiaddress",
// 		Flags: []cli.Flag{
// 			&cli.DurationFlag{
// 				Name:    "timeout",
// 				Aliases: []string{"t"},
// 				Usage:   "timeout for discovering peers",
// 				Value:   5 * time.Second,
// 				EnvVars: []string{"TIMEOUT"},
// 			},
// 			&cli.IntFlag{
// 				Name:    "num",
// 				Aliases: []string{"n"},
// 				Usage:   "amount of maximum peers desired to discover",
// 				Value:   1,
// 				EnvVars: []string{"PEERS_NUM"},
// 			},
// 			&cli.BoolFlag{
// 				Name:    "json",
// 				Usage:   "print results as json",
// 				Value:   false,
// 				EnvVars: []string{"OUTPUT_JSON"},
// 			},
// 		},
// 		Action: discover,
// 	}
// }

// func discover(c *cli.Context) error {
// 	logger = logutil.New(c).
// 		WithField("limit", c.Int("num"))

// 	h, err := libp2p.New(
// 		libp2p.NoTransports,
// 		libp2p.NoListenAddrs,
// 		libp2p.Transport(quic.NewTransport))
// 	if err != nil {
// 		return err
// 	}

// 	discoverer, err := bootutil.DialString(h, c.String("discover"),
// 		socket.WithLogger(logger), socket.WithRateLimiter(socket.NewPacketLimiter(1000, 8)))
// 	if err != nil {
// 		return err
// 	}

// 	ctx, cancel := context.WithTimeout(c.Context, c.Duration("timeout"))
// 	defer cancel()

// 	infos, err := discoverer.FindPeers(ctx, c.String("ns"),
// 		discovery.Limit(c.Int("num")))
// 	if err != nil {
// 		return err
// 	}

// 	for info := range infos {
// 		as, err := peer.AddrInfoToP2pAddrs(&info)
// 		if err != nil {
// 			return err
// 		}

// 		info.Addrs = as

// 		print := printer(c)

// 		if err = print(info); err != nil {
// 			return err
// 		}
// 	}

// 	return ctx.Err()
// }

// func printer(c *cli.Context) func(peer.AddrInfo) error {
// 	if c.Bool("json") {
// 		return jsonPrinter(c)
// 	}

// 	return textPrinter(c)
// }

// func jsonPrinter(c *cli.Context) func(peer.AddrInfo) error {
// 	enc := json.NewEncoder(c.App.Writer)

// 	return func(info peer.AddrInfo) error {
// 		return enc.Encode(info)
// 	}
// }

// func textPrinter(c *cli.Context) func(peer.AddrInfo) error {
// 	return func(info peer.AddrInfo) error {
// 		_, err := fmt.Fprintln(c.App.Writer, info)
// 		return err
// 	}
// }
