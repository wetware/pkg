package runtime

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/metrics"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	disc "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/thejerf/suture/v4"

	"github.com/lthibault/log"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"github.com/wetware/casm/pkg/boot"
	"github.com/wetware/casm/pkg/boot/crawl"
	"github.com/wetware/casm/pkg/pex"
	bootutil "github.com/wetware/ww/internal/util/boot"
	statsdutil "github.com/wetware/ww/internal/util/statsd"
	"github.com/wetware/ww/pkg/vat"
)

var network = fx.Provide(
	bootstrap,
	vatnet,
	overlay,
	beacon)

type networkModule struct {
	fx.Out

	Vat vat.Network
	DHT *dual.DHT
}

func vatnet(c *cli.Context, lx fx.Lifecycle, b *metrics.BandwidthCounter) (mod networkModule, err error) {
	mod.Vat.NS = c.String("ns")
	mod.Vat.Host, err = libp2p.New(
		libp2p.NoTransports,
		libp2p.Transport(libp2pquic.NewTransport),
		libp2p.ListenAddrStrings(c.StringSlice("listen")...),
		libp2p.BandwidthReporter(b))
	if err != nil {
		return
	}

	lx.Append(closer(mod.Vat.Host))

	mod.DHT, err = dual.New(c.Context, mod.Vat.Host,
		dual.LanDHTOption(dht.Mode(dht.ModeServer)),
		dual.WanDHTOption(dht.Mode(dht.ModeAuto)))
	if err != nil {
		return
	}

	lx.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return mod.DHT.Bootstrap(ctx)
		},
		OnStop: func(context.Context) error {
			return mod.DHT.Close()
		},
	})

	mod.Vat.Host = routedhost.Wrap(mod.Vat.Host, mod.DHT)
	return
}

func overlay(c *cli.Context, vat vat.Network, d discovery.Discovery) (*pubsub.PubSub, error) {
	return pubsub.NewGossipSub(c.Context, vat.Host,
		pubsub.WithRawTracer(statsdutil.NewPubSubTracer(c)),
		pubsub.WithDiscovery(d))
}

type bootstrapConfig struct {
	fx.In

	Log        log.Logger
	Vat        vat.Network
	Datastore  ds.Batching
	DHT        *dual.DHT
	Supervisor *suture.Supervisor

	Lifecycle fx.Lifecycle
}

func (config bootstrapConfig) Logger() log.Logger {
	return config.Log.With(config.Vat)
}

func bootstrap(c *cli.Context, config bootstrapConfig) (discovery.Discovery, error) {
	d, err := bootutil.New(c, config.Vat.Host)
	if err != nil {
		return nil, err
	}

	var b = bootstrapper{
		Log:        config.Logger(),
		Discoverer: d,
	}

	switch x := d.(type) {
	case discovery.Advertiser:
		b.Advertiser = x

	case crawl.Crawler:
		a, err := beacon(c, b.Log, config.Vat)
		if err != nil {
			err = fmt.Errorf("beacon: %w", err)
		}

		config.Lifecycle.Append(fx.Hook{
			OnStart: func(context.Context) error {
				config.Supervisor.Add(a)
				return nil
			},
		})

		b.Advertiser = a
	}

	// Wrap the bootstrap discovery service in a peer sampling service.
	px, err := pex.New(config.Vat.Host,
		pex.WithLogger(config.Logger()),
		pex.WithDatastore(config.Datastore),
		pex.WithDiscovery(b))
	if err != nil {
		return nil, err
	}
	config.Lifecycle.Append(closer(px))

	// If the namespace matches the cluster pubsub topic,
	// fetch peers from PeX, which itself will fall back
	// on the bootstrap services.
	return boot.Namespace{
		Match:   pubsubTopic(config.Vat.NS),
		Target:  px,
		Default: disc.NewRoutingDiscovery(config.DHT),
	}, nil
}

type bootstrapper struct {
	Log log.Logger
	discovery.Discoverer
	discovery.Advertiser
}

func (b bootstrapper) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	b.Log.Debug("bootstrapping namespace")
	return b.Discoverer.FindPeers(ctx, ns, opt...)
}

func (b bootstrapper) Advertise(ctx context.Context, ns string, opt ...discovery.Option) (time.Duration, error) {
	if b.Advertiser == nil {
		return peerstore.PermanentAddrTTL, nil
	}

	b.Log.Debug("advertising namespace")
	return b.Advertiser.Advertise(ctx, ns, opt...)
}

func beacon(c *cli.Context, log log.Logger, vat vat.Network) (crawl.Beacon, error) {
	addr, err := cidrToListenAddr(c)
	return crawl.Beacon{
		Logger: log.WithField("beacon", c.String("discover")),
		Addr:   addr,
		Host:   vat.Host,
	}, err
}

func cidrToListenAddr(c *cli.Context) (net.Addr, error) {
	maddr, err := ma.NewMultiaddr(c.String("discover"))
	if err != nil {
		return nil, err
	}

	network, addr, err := manet.DialArgs(maddr)
	if err != nil {
		return nil, err
	}

	switch network {
	case "tcp", "tcp4", "tcp6":
		return net.ResolveTCPAddr(network, addr)

	case "udp", "udp4", "udp6":
		return net.ResolveUDPAddr(network, addr)

	default:
		return nil, fmt.Errorf("invalid network: %s", network)
	}
}

func pubsubTopic(match string) func(string) bool {
	const prefix = "floodsub:"

	return func(s string) bool {
		return match == strings.TrimPrefix(s, prefix)
	}
}
