package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/peer"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
	bootutil "github.com/wetware/casm/pkg/boot/util"
)

func Discover() *cli.Command {
	return &cli.Command{
		Name:  "discover",
		Usage: "discover a wetware node and print its multiaddress",
		Flags: []cli.Flag{
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
	h, err := libp2p.New(
		libp2p.NoTransports,
		libp2p.NoListenAddrs,
		libp2p.Transport(libp2pquic.NewTransport))
	if err != nil {
		return err
	}

	discoverer, err := bootutil.DialString(h, c.String("discover"))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(c.Context, time.Duration(c.Duration("timeout")))
	defer cancel()

	infos, err := discoverer.FindPeers(ctx, c.String("ns"))
	if err != nil {
		return err
	}

	discovered := make([]peer.AddrInfo, 0, c.Int("amount"))
	for i := 0; i < c.Int("amount"); i++ {
		select {
		case info := <-infos:
			err := setP2pAddress(info)
			if err != nil {
				return err
			}
			discovered = append(discovered, info)
		case <-ctx.Done():
		}
	}

	// print results
	if c.Bool("json") {
		jsonOutput, err := json.Marshal(discovered)
		if err != nil {
			return nil
		}
		fmt.Println(string(jsonOutput))
	} else {
		for _, info := range discovered {
			fmt.Println(info.String())
		}
	}

	return ctx.Err()
}

func setP2pAddress(info peer.AddrInfo) error {
	for i := range info.Addrs {
		maddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", info.ID.String()))
		if err != nil {
			return err
		}
		info.Addrs[i] = info.Addrs[i].Encapsulate(maddr)
	}
	return nil
}
