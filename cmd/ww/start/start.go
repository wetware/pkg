package start

import (
	"fmt"

	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"

	ww "github.com/wetware/pkg"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/boot"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/client"
	"github.com/wetware/pkg/server"
	"github.com/wetware/pkg/util/proto"
)

var flags = []cli.Flag{
	&cli.StringSliceFlag{
		Name:    "listen",
		Aliases: []string{"l"},
		Usage:   "host listen address",
		Value: cli.NewStringSlice(
			"/ip4/0.0.0.0/udp/0/quic-v1",
			"/ip6/::0/udp/0/quic-v1"),
		EnvVars: []string{"WW_LISTEN"},
	},
	&cli.StringSliceFlag{
		Name:    "meta",
		Usage:   "metadata fields in key=value format",
		EnvVars: []string{"WW_META"},
	},
}

func Command() *cli.Command {
	return &cli.Command{
		Name:   "start",
		Usage:  "start a host process",
		Flags:  flags,
		Action: serve,
	}
}

func serve(c *cli.Context) error {
	h, err := server.ListenP2P(c.StringSlice("listen")...)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer h.Close()

	d, err := newDiscovery(c, h)
	if err != nil {
		return fmt.Errorf("discovery: %w", err)
	}
	defer d.Close()

	boot := server.BootConfig{
		NS:        c.String("ns"),
		Host:      h,
		Discovery: d,
	}

	dialer := client.Dialer[host.Host]{
		Bootstrapper: boot,
		Auth:         auth.AllowAll[host.Host],
		Opts:         nil, // TODO:  export something from the client side
	}

	server := server.Config{
		NS:    boot.NS,
		Proto: proto.Namespace(boot.NS),
		Meta:  c.StringSlice("meta"),
		Host:  h,
		Boot:  boot,
		Auth:  auth.AllowAll[host.Host],
	}

	return ww.Vat{
		Addr:   addr(c, h),
		Dialer: dialer,
		// Export: export,
		Server: server,
	}.ListenAndServe(c.Context)
}

// local address
func addr(c *cli.Context, h local.Host) *ww.Addr {
	ns := c.String("ns")
	return &ww.Addr{
		NS:    ns,
		Peer:  h.ID(),
		Proto: proto.Namespace(ns),
	}
}

func newDiscovery(c *cli.Context, h local.Host) (_ boot.Service, err error) {
	// use discovery service?
	if len(c.StringSlice("peer")) == 0 {
		serviceAddr := c.String("discover")
		return boot.ListenString(h, serviceAddr)
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
