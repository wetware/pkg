package cluster

import (
	"fmt"

	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"

	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/boot"
	"github.com/wetware/pkg/vat"
)

type CloseFunc func() error

func Command() *cli.Command {
	return &cli.Command{
		Name:    "cluster",
		Usage:   "cli client for wetware clusters",
		Aliases: []string{"client"}, // TODO(soon):  deprecate
		Subcommands: []*cli.Command{
			run(),
		},
	}
}

// Login in into the cluster and get an auth.Session capability.
func BootstrapSession(c *cli.Context, h local.Host) (s auth.Session, r CloseFunc, err error) {
	// Connect to peers.
	bootstrap, err := newBootstrap(c, h)
	if err != nil {
		err = fmt.Errorf("discovery: %w", err)
		return
	}
	r = func() error {
		defer h.Close()
		defer bootstrap.Close()
		return nil
	}

	// Login into the wetware cluster.
	s, err = vat.Dialer{
		Host:    h,
		Account: auth.SignerFromHost(h),
	}.DialDiscover(c.Context, bootstrap, c.String("ns"))
	if err != nil {
		return
	}
	return
}

func newBootstrap(c *cli.Context, h local.Host) (_ boot.Service, err error) {
	// use discovery service?
	if len(c.StringSlice("peer")) == 0 {
		serviceAddr := c.String("discover")
		return boot.DialString(h, serviceAddr)
	}

	// fast path; direct dial a peer
	maddrs := make([]ma.Multiaddr, len(c.StringSlice("peer")))
	for i, s := range c.StringSlice("peer") {
		if maddrs[i], err = ma.NewMultiaddr(s); err != nil {
			return
		}
	}

	infos, err := peer.AddrInfosFromP2pAddrs(maddrs...)
	return boot.StaticAddrs(infos), err
}
