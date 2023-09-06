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

var (
	session auth.Session
	// releases *[]func()
	// closes   *[]func() error
)

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

func setup() cli.BeforeFunc {
	return func(c *cli.Context) (err error) {
		h, err := vat.DialP2P()
		if err != nil {
			return err
		}
		defer h.Close()

		bootstrap, err := newBootstrap(c, h)
		if err != nil {
			return fmt.Errorf("discovery: %w", err)
		}
		defer bootstrap.Close()

		session, err = vat.Dialer{
			Host:    h,
			Account: auth.SignerFromHost(h),
		}.DialDiscover(c.Context, bootstrap, c.String("ns"))
		return err
	}
}

func teardown() cli.AfterFunc {
	return func(c *cli.Context) (err error) {
		// for _, close := range *closes {
		// 	defer close()
		// }
		// for _, release := range *releases {
		// 	defer release()
		// }
		return nil
	}
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
