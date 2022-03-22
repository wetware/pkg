package runtime

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
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
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/vat"
)

var network = fx.Provide(
	vatnet,
	overlay,
	discover,
	bootstrap,
	peercache)

type routingConfig struct {
	fx.In

	CLI       *cli.Context
	Metrics   *metrics.BandwidthCounter
	Lifecycle fx.Lifecycle
}

func (config routingConfig) ListenAddrs() []string {
	return config.CLI.StringSlice("listen")
}

func (config routingConfig) NewHost() (h host.Host, err error) {
	h, err = libp2p.New(
		libp2p.NoTransports,
		libp2p.Transport(libp2pquic.NewTransport),
		libp2p.ListenAddrStrings(config.ListenAddrs()...),
		libp2p.BandwidthReporter(config.Metrics))
	if err == nil {
		config.Lifecycle.Append(closer(h))
	}

	return
}

func (config routingConfig) LANOpt() []dht.Option {
	return []dht.Option{
		dht.Mode(dht.ModeServer),
		dht.ProtocolPrefix(ww.Subprotocol(config.CLI.String("ns"))),
		dht.ProtocolExtension("lan")}
}

func (config routingConfig) WANOpt() []dht.Option {
	return []dht.Option{
		dht.Mode(dht.ModeAuto),
		dht.ProtocolPrefix(ww.Subprotocol(config.CLI.String("ns"))),
		dht.ProtocolExtension("wan")}
}

func (config routingConfig) NewDHT(h host.Host) (*dual.DHT, error) {
	// TODO:  Use dht.BootstrapPeersFunc to get bootstrap peers from PeX?
	//        This might allow us to greatly simplify our architecture and
	//        runtime initialization.  In particular:
	//
	//          1. The DHT could query PeX directly, eliminating the need for
	//             dynamic dispatch via boot.Namespace.
	//
	//          2. The server.Joiner type could be simplified, and perhaps
	//             eliminated entirely.

	d, err := dual.New(config.CLI.Context, h,
		dual.LanDHTOption(config.LANOpt()...),
		dual.WanDHTOption(config.WANOpt()...))

	if err == nil {
		config.Lifecycle.Append(fx.Hook{
			OnStart: d.Bootstrap,
			OnStop:  onclose(d),
		})
	}

	return d, err
}

func routing(config routingConfig) (*dual.DHT, error) {
	h, err := config.NewHost()
	if err != nil {
		return nil, err
	}

	return config.NewDHT(h)
}

type vatConfig struct {
	fx.In

	CLI       *cli.Context
	DHT       *dual.DHT
	Lifecycle fx.Lifecycle
}

func (vat vatConfig) Namespace() string {
	return vat.CLI.String("ns")
}

func (vat vatConfig) Host() host.Host {
	return vat.DHT.WAN.Host()
}

func vatnet(config vatConfig) vat.Network {
	return vat.Network{
		NS:   config.Namespace(),
		Host: routedhost.Wrap(config.Host(), config.DHT),
	}
}

func overlay(c *cli.Context, vat vat.Network, d discovery.Discovery) (*pubsub.PubSub, error) {
	return pubsub.NewGossipSub(c.Context, vat.Host,
		pubsub.WithRawTracer(statsdutil.NewPubSubTracer(c)),
		pubsub.WithDiscovery(d))
}

type discoveryConfig struct {
	fx.In

	Vat vat.Network
	PeX *pex.PeerExchange
	DHT *dual.DHT
}

func (config discoveryConfig) matchNS() func(string) bool {
	bootTopic := "floodsub:" + config.Vat.NS
	return func(ns string) bool {
		return ns == bootTopic
	}
}

// discover constructs the top-level discovery service, which dynamically
// dispatches advertisements and search queries to either:
//
// 1. the bootstrap service, iff the namespace matches the cluster topic; else
// 2. the DHT-backed ambient peer discovery service.
func discover(config discoveryConfig) (discovery.Discovery, error) {
	// If the namespace matches the cluster pubsub topic,
	// fetch peers from PeX, which itself will fall back
	// on the bootstrap services.
	return boot.Namespace{
		Match:   config.matchNS(),
		Target:  config.PeX,
		Default: disc.NewRoutingDiscovery(config.DHT),
	}, nil
}

type pexConfig struct {
	fx.In

	Log       log.Logger
	Vat       vat.Network
	Datastore ds.Batching
	Boot      bootstrapper
	Lifecycle fx.Lifecycle
}

func (config pexConfig) Host() host.Host {
	return config.Vat.Host
}

func (config pexConfig) Logger() log.Logger {
	return config.Log.With(config.Vat)
}

func (config pexConfig) SetCloseHook(c io.Closer) {
	config.Lifecycle.Append(closer(c))
}

func peercache(config pexConfig) (*pex.PeerExchange, error) {
	px, err := pex.New(config.Host(),
		pex.WithLogger(config.Logger()),
		pex.WithDatastore(config.Datastore),
		pex.WithDiscovery(config.Boot))

	if err == nil {
		config.SetCloseHook(px)
	}

	return px, err
}

type bootConfig struct {
	fx.In

	CLI        *cli.Context
	Log        log.Logger
	Vat        vat.Network
	Supervisor *suture.Supervisor
	Lifecycle  fx.Lifecycle
}

func (config bootConfig) Logger() log.Logger {
	return config.Log.With(config.Vat)
}

func (config bootConfig) Host() host.Host {
	return config.Vat.Host
}

func (config bootConfig) NewBeacon() (b crawl.Beacon, err error) {
	if b.Addr, err = cidrToListenAddr(config.CLI); err == nil {
		b.Logger = config.Logger().WithField("beacon", b.Addr)
		b.Host = config.Vat.Host
		config.AddService(b)
	}

	return
}

func (config bootConfig) AddService(s suture.Service) {
	config.Lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			config.Supervisor.Add(s)
			return nil
		},
	})
}

type bootstrapper struct {
	Log log.Logger
	discovery.Discoverer
	discovery.Advertiser
}

func bootstrap(config bootConfig) (b bootstrapper, err error) {
	b.Log = config.Logger()
	b.Discoverer, err = bootutil.New(config.CLI, config.Host())

	if err == nil {
		switch x := b.Discoverer.(type) {
		case discovery.Advertiser:
			b.Advertiser = x

		case crawl.Crawler:
			b.Advertiser, err = config.NewBeacon()
		}
	}

	return
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
