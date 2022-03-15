package bootutil

import (
	"errors"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/wetware/casm/pkg/boot"
	"github.com/wetware/ww/pkg/client"
	"github.com/wetware/ww/pkg/vat"

	"github.com/urfave/cli/v2"
)

func New(c *cli.Context, h host.Host) (discovery.Discoverer, error) {
	if c.IsSet("addr") {
		return boot.NewStaticAddrStrings(c.StringSlice("addr")...)
	}

	if c.String("discover") == "" {
		return nil, errors.New("must provide -discover or -addr flag")
	}

	addr, err := ma.NewMultiaddr(c.String("discover"))
	if err != nil {
		return nil, err
	}

	return boot.Parse(h, addr)
}

func Dial(c *cli.Context, h host.Host) (*client.Node, error) {
	b, err := New(c, h)
	if err != nil {
		return nil, err
	}

	return client.Dialer{
		Boot: b,
		Vat: vat.Network{
			NS:   c.String("ns"),
			Host: h,
		},
	}.Dial(c.Context)
}
