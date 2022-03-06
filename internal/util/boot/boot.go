package bootutil

import (
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/wetware/casm/pkg/boot"
	"github.com/wetware/ww/pkg/vat"

	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
)

func NewDiscovery(c *cli.Context, vat vat.Network) (discovery.Discoverer, error) {
	maddr, err := multiaddr.NewMultiaddr(c.String("discover"))
	if err != nil {
		return nil, err
	}
	return boot.Parse(vat.Host, maddr)
}
