package start

import (
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/discovery"
	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	disc_util "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/urfave/cli/v2"

	core_api "github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/boot"
	csp_server "github.com/wetware/pkg/cap/csp/server"
	"github.com/wetware/pkg/cluster/pulse"
	"github.com/wetware/pkg/cluster/routing"
	"github.com/wetware/pkg/vat"
)

var meta tags

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
		Before: setup,
		Action: serve,
	}
}

func setup(c *cli.Context) error {
	deduped := make(map[routing.MetaField]struct{})
	for _, tag := range c.StringSlice("meta") {
		field, err := routing.ParseField(tag)
		if err != nil {
			return err
		}

		deduped[field] = struct{}{}
	}

	for tag := range deduped {
		meta = append(meta, tag)
	}

	return nil
}

func serve(c *cli.Context) error {
	h, err := vat.ListenP2P(c.StringSlice("listen")...)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer h.Close()

	dht, err := vat.NewDHT(c.Context, h, c.String("ns"))
	if err != nil {
		return fmt.Errorf("dht: %w", err)
	}
	defer dht.Close()

	bootstrap, err := newBootstrap(c, h)
	if err != nil {
		return fmt.Errorf("discovery: %w", err)
	}
	defer bootstrap.Close()

	ec := make(chan csp_server.Runtime, 1)
	sc := make(chan core_api.Session, 1)
	return vat.Config{
		NS:        c.String("ns"),
		Host:      routedhost.Wrap(h, dht),
		Bootstrap: bootstrap,
		Ambient:   ambient(dht),
		Meta:      meta,
		Auth:      auth.AllowAll,
	}.Serve(c.Context, ec, sc, h)
}

func newBootstrap(c *cli.Context, h local.Host) (_ boot.Service, err error) {
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

func ambient(dht *dual.DHT) discovery.Discovery {
	return disc_util.NewRoutingDiscovery(dht)
}

type tags []routing.MetaField

func (tags tags) Prepare(h pulse.Heartbeat) error {
	if err := h.SetMeta(tags); err != nil {
		return err
	}

	// hostname may change over time
	host, err := os.Hostname()
	if err != nil {
		return err
	}

	return h.SetHost(host)
}
