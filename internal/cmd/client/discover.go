package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/peer"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
	"github.com/wetware/casm/pkg/boot/socket"
	bootutil "github.com/wetware/casm/pkg/boot/util"
	logutil "github.com/wetware/ww/internal/util/log"
)

func Discover() *cli.Command {
	return &cli.Command{
		Name:  "discover",
		Usage: "discover a wetware node and print its multiaddress",
		Flags: []cli.Flag{
			&cli.DurationFlag{
				Name:    "timeout",
				Aliases: []string{"t"},
				Usage:   "timeout for discovering peers",
				Value:   5 * time.Second,
				EnvVars: []string{"TIMEOUT"},
			},
			&cli.IntFlag{
				Name:    "num",
				Aliases: []string{"n"},
				Usage:   "amount of maximum peers desired to discover",
				Value:   1,
				EnvVars: []string{"PEERS_NUM"},
			},
			&cli.BoolFlag{
				Name:    "json",
				Usage:   "print results as json",
				Value:   false,
				EnvVars: []string{"OUTPUT_JSON"},
			},
		},
		Action: discover,
	}
}

func discover(c *cli.Context) error {
	logger = logutil.New(c).
		WithField("limit", c.Int("num"))

	h, err := libp2p.New(
		libp2p.NoTransports,
		libp2p.NoListenAddrs,
		libp2p.Transport(libp2pquic.NewTransport))
	if err != nil {
		return err
	}

	discoverer, err := bootutil.DialString(h, c.String("discover"),
		socket.WithLogger(logger))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(c.Context, c.Duration("timeout"))
	defer cancel()

	infos, err := discoverer.FindPeers(ctx, c.String("ns"),
		discovery.Limit(c.Int("num")))
	if err != nil {
		return err
	}

	for info := range infos {
		as, err := peer.AddrInfoToP2pAddrs(&info)
		if err != nil {
			return err
		}

		print := printer(c)
		for _, addr := range as {
			if err = print(addr); err != nil {
				return err
			}
		}
	}

	return ctx.Err()
}

func printer(c *cli.Context) func(multiaddr.Multiaddr) error {
	if c.Bool("json") {
		return jsonPrinter(c)
	}

	return textPrinter(c)
}

func jsonPrinter(c *cli.Context) func(multiaddr.Multiaddr) error {
	enc := json.NewEncoder(c.App.Writer)

	return func(maddr multiaddr.Multiaddr) error {
		return enc.Encode(maddr)
	}
}

func textPrinter(c *cli.Context) func(multiaddr.Multiaddr) error {
	return func(maddr multiaddr.Multiaddr) error {
		_, err := fmt.Fprintln(c.App.Writer, maddr)
		return err
	}
}

// func setP2pAddress(info peer.AddrInfo) error {
// 	for i := range info.Addrs {
// 		maddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", info.ID.String()))
// 		if err != nil {
// 			return err
// 		}
// 		info.Addrs[i] = info.Addrs[i].Encapsulate(maddr)
// 	}
// 	return nil
// }
