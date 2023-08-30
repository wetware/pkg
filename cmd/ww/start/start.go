package start

import (
	"fmt"
	"os"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/discovery"
	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	disc_util "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/urfave/cli/v2"

	"github.com/wetware/pkg/boot"
	"github.com/wetware/pkg/cluster/pulse"
	"github.com/wetware/pkg/cluster/routing"
	"github.com/wetware/pkg/server"
	"github.com/wetware/pkg/util/proto"
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
	h, err := server.ListenP2P(c.StringSlice("listen")...)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer h.Close()

	dht, err := newDHT(c, h)
	if err != nil {
		return fmt.Errorf("dht: %w", err)
	}
	defer dht.Close()

	bootstrap, err := newBootstrap(c, h)
	if err != nil {
		return fmt.Errorf("discovery: %w", err)
	}
	defer bootstrap.Close()

	ns := boot.Namespace{
		Name:      c.String("ns"),
		Bootstrap: bootstrap,
		Ambient:   ambient(dht),
	}

	return server.Vat{
		NS:   ns,
		Host: routedhost.Wrap(h, dht),
		Meta: meta,
	}.Serve(c.Context)
}

func newDHT(c *cli.Context, h local.Host) (*dual.DHT, error) {
	ns := c.String("ns")
	return dual.New(c.Context, h,
		dual.LanDHTOption(lanOpt(ns)...),
		dual.WanDHTOption(wanOpt(ns)...))
}

func lanOpt(ns string) []dht.Option {
	return []dht.Option{
		dht.Mode(dht.ModeServer),
		dht.ProtocolPrefix(proto.Root(ns)),
		dht.ProtocolExtension("lan")}
}

func wanOpt(ns string) []dht.Option {
	return []dht.Option{
		dht.Mode(dht.ModeAuto),
		dht.ProtocolPrefix(proto.Root(ns)),
		dht.ProtocolExtension("wan")}
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
