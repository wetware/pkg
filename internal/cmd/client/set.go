package client

import (
	"errors"
	"strings"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/urfave/cli/v2"
	"github.com/wetware/casm/pkg/boot"
	"github.com/wetware/ww/pkg/cap/cluster"
	"github.com/wetware/ww/pkg/client"
	"github.com/wetware/ww/pkg/vat"
)

func Set() *cli.Command {
	var (
		d discovery.Discoverer
		v vat.Network
	)

	return &cli.Command{
		Name:   "set",
		Usage:  "set data in cluster path",
		Flags:  clientFlags,
		Before: beforeClient(&d, &v),
		Action: set(&d, &v),
		After:  afterClient(&v),
	}
}

func set(d *discovery.Discoverer, v *vat.Network) cli.ActionFunc {
	return func(c *cli.Context) error {
		peers, err := (*d).FindPeers(c.Context, c.String("ns"))
		if err != nil {
			return err
		}

		var n *client.Node

		for info := range peers {
			n, err = client.Dialer{
				Vat:  (*v),
				Boot: boot.StaticAddrs{info},
			}.Dial(c.Context)
			if err != nil {
				return err
			}
			break
		}

		if n == nil {
			return errors.New("no server found")
		}

		path := cleanPath(strings.Split(c.Args().First(), "/"))
		a, err := n.Walk(c.Context, path)
		if err != nil {
			return err
		}
		defer a.Release(c.Context)

		co, ok := a.(cluster.Container)
		if !ok {
			return errors.New("path is not settable")
		}

		return co.Set(c.Context, []byte(c.Args().Get(1)))
	}
}
