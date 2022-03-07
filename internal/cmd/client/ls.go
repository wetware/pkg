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
	var (
		v vat.Network
		n *client.Node
	)

	return &cli.Command{
		Name:   "ls",
		Usage:  "list information about cluster path",
		Flags:  clientFlags,
		Before: beforeAnchor(&v, &n),
		Action: ls(&n),
		After:  afterAnchor(&v),
	}
}

var clientFlags = []cli.Flag{
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
			"/ip4/0.0.0.0/tcp/0/quic",
			"/ip6/::0/udp/0/quic"),
		EnvVars: []string{"WW_LISTEN"},
	},
	&cli.StringFlag{
		Name:    "discover",
		Aliases: []string{"d"},
		Usage:   "bootstrap discovery addr (cidr url)",
		Value:   "/ip4/228.8.8.8/udp/8822/survey", // TODO:  this should default to survey
		EnvVars: []string{"WW_DISCOVER"},
	},
}

func beforeAnchor(v *vat.Network, n **client.Node) cli.BeforeFunc {
	return func(c *cli.Context) error {
		h, err := libp2p.New(c.Context,
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

		*v = vat.Network{
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
			*n, err = client.Dialer{
				Vat:  (*v),
				Boot: boot.StaticAddrs{info},
			}.Dial(c.Context)
			if err != nil {
				return err
			}
			break
		}
		if *n == nil {
			return errors.New("no server found")
		}
		time.Sleep(100 * time.Millisecond) // add delay to propagate the peer to the DHT table
		return nil
	}
}

func ls(nn **client.Node) cli.ActionFunc {
	return func(c *cli.Context) error {
		n := *nn
		path := cleanPath(strings.Split(c.Args().First(), "/"))
		anchor, err := n.Walk(c.Context, path)
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

func afterAnchor(v *vat.Network) cli.AfterFunc {
	return func(ctx *cli.Context) error {
		if (*v).Host != nil {
			return v.Host.Close()
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
