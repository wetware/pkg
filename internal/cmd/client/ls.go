package client

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
	"github.com/wetware/casm/pkg/boot"
	"github.com/wetware/ww/pkg/client"
	"github.com/wetware/ww/pkg/vat"
)

func Ls() *cli.Command {
	return &cli.Command{
		Name:   "ls",
		Usage:  "list information about cluster path",
		Flags:  clientFlags,
		Before: beforeAnchor(),
		Action: ls(),
		After:  afterAnchor(),
	}
}

var clientFlags = []cli.Flag{}

func beforeAnchor() cli.BeforeFunc {
	return func(c *cli.Context) error {
		var err error

		h, err = libp2p.New(c.Context,
			libp2p.DefaultTransports,
			libp2p.Transport(libp2pquic.NewTransport),
			libp2p.ListenAddrStrings(c.StringSlice("listen")...))

		if err != nil {
			return err
		}

		dht, err := dual.New(c.Context, h,
			dual.LanDHTOption(dht.Mode(dht.ModeServer)),
			dual.WanDHTOption(dht.Mode(dht.ModeAuto)))
		if err != nil {
			return err
		}
		if err := dht.Bootstrap(c.Context); err != nil {
			return err
		}

		h = routedhost.Wrap(h, dht)

		v := vat.Network{
			NS:   c.String("ns"),
			Host: h,
		}

		maddr, err := multiaddr.NewMultiaddr(c.String("discover"))
		if err != nil {
			return err
		}

		d, err := boot.Parse(h, maddr)
		if err != nil {
			return err
		}

		peers, err := d.FindPeers(c.Context, c.String("ns"))
		if err != nil {
			return err
		}

		for info := range peers {
			node, err = client.Dialer{
				Vat:  v,
				Boot: boot.StaticAddrs{info},
			}.Dial(c.Context)
			if err != nil {
				return err
			}
			break
		}
		if node == nil {
			return errors.New("no server found")
		}
		time.Sleep(100 * time.Millisecond) // add delay to propagate the peer to the DHT table
		return nil
	}
}

func ls() cli.ActionFunc {
	return func(c *cli.Context) error {
		path := cleanPath(strings.Split(c.Args().First(), "/"))
		anchor, err := node.Walk(c.Context, path)
		if err != nil {
			return err
		}
		defer anchor.Release(c.Context)

		it, err := anchor.Ls(c.Context)
		if err != nil {
			return err
		}

		defer it.Finish()

		for it.Next(c.Context) {
			fmt.Printf("/%v\n", it.Anchor().Name())
		}

		return nil
	}
}

func afterAnchor() cli.AfterFunc {
	return func(ctx *cli.Context) error {
		if h != nil {
			return h.Close()
		}
		return nil
	}
}

func cleanPath(path []string) (newPath []string) {
	newPath = path[:0]
	for _, e := range path {
		if e != "" {
			newPath = append(newPath, e)
		}
	}
	return
}
