package runtime

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"strings"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/metrics"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"

	"github.com/lthibault/log"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/boot"
	"github.com/wetware/casm/pkg/boot/socket"
	bootutil "github.com/wetware/casm/pkg/boot/util"
	"github.com/wetware/casm/pkg/pex"
	protoutil "github.com/wetware/casm/pkg/util/proto"
	statsdutil "github.com/wetware/ww/internal/util/statsd"
	ww "github.com/wetware/ww/pkg"
)

var network = fx.Provide(
	vatnet,
	overlay,
	bootstrap,
	peercache,
	dhtRouting,
	randomIdentityEd25519,
)

type routingConfig struct {
	fx.In

	CLI       *cli.Context
	Metrics   *metrics.BandwidthCounter
	Lifecycle fx.Lifecycle
	Priv      crypto.PrivKey
}

func dhtRouting(config routingConfig) (*dual.DHT, error) {
	h, err := config.NewHost()
	if err != nil {
		return nil, err
	}

	return config.NewDHT(h)
}

func (config routingConfig) ListenAddrs() []string {
	return config.CLI.StringSlice("listen")
}

func (config routingConfig) NewHost() (h host.Host, err error) {
	h, err = libp2p.New(
		libp2p.NoTransports,
		libp2p.Transport(quic.NewTransport),
		libp2p.ListenAddrStrings(config.ListenAddrs()...),
		libp2p.BandwidthReporter(config.Metrics),
		libp2p.Identity(config.Priv))
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

type vatConfig struct {
	fx.In

	CLI       *cli.Context
	DHT       *dual.DHT
	Lifecycle fx.Lifecycle
	Metrics   *statsdutil.MetricsReporter
}

func vatnet(config vatConfig) casm.Vat {
	return casm.Vat{
		NS:      config.Namespace(),
		Host:    routedhost.Wrap(config.Host(), config.DHT),
		Metrics: config.Metrics.NewStore(),
	}
}

func (vat vatConfig) Namespace() string {
	return vat.CLI.String("ns")
}

func (vat vatConfig) Host() host.Host {
	return vat.DHT.WAN.Host()
}

type pexConfig struct {
	fx.In

	Log       log.Logger
	Vat       casm.Vat
	Datastore ds.Batching
	Boot      bootstrapper
	Lifecycle fx.Lifecycle
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

func (config pexConfig) Host() host.Host {
	return config.Vat.Host
}

func (config pexConfig) Logger() log.Logger {
	return config.Log.With(config.Vat)
}

func (config pexConfig) SetCloseHook(c io.Closer) {
	config.Lifecycle.Append(closer(c))
}

type bootConfig struct {
	fx.In

	CLI       *cli.Context
	Log       log.Logger
	Vat       casm.Vat
	Lifecycle fx.Lifecycle
}

func (config bootConfig) Logger() log.Logger {
	return config.Log.With(config.Vat)
}

func (config bootConfig) Host() host.Host {
	return config.Vat.Host
}

func (config bootConfig) SetCloseHook(c io.Closer) {
	config.Lifecycle.Append(closer(c))
}

type bootstrapper struct {
	Log log.Logger
	discovery.Discovery
}

func bootstrap(config bootConfig) (b bootstrapper, err error) {
	b.Log = config.Logger()

	if config.CLI.IsSet("addr") {
		b.Discovery, err = boot.NewStaticAddrStrings(config.CLI.StringSlice("addr")...)
	} else {
		b.Discovery, err = bootutil.ListenString(config.Host(), config.CLI.String("discover"), socket.WithRateLimiter(socket.NewPacketLimiter(1000, 8)))
	}

	if err == nil {
		if c, ok := b.Discovery.(io.Closer); ok {
			config.SetCloseHook(c)
		}
	}
	return
}

func (b bootstrapper) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	b.Log.Debug("bootstrapping namespace")
	return b.Discovery.FindPeers(ctx, strings.TrimPrefix(ns, "floodsub:"), opt...)
}

func (b bootstrapper) Advertise(ctx context.Context, ns string, opt ...discovery.Option) (time.Duration, error) {
	b.Log.Debug("advertising namespace")
	return b.Discovery.Advertise(ctx, strings.TrimPrefix(ns, "floodsub:"), opt...)
}

type overlayConfig struct {
	fx.In

	CLI *cli.Context
	Vat casm.Vat
	// PeX    *pex.PeerExchange  // TODO:  re-enable when PeX bugs are fixed
	Boot   bootstrapper // TODO:  remove when PeX bugs are fixed
	DHT    *dual.DHT
	Tracer *statsdutil.PubSubTracer
}

func overlay(config overlayConfig) (*pubsub.PubSub, error) {
	return pubsub.NewGossipSub(config.Context(), config.Host(),
		pubsub.WithPeerExchange(true),
		pubsub.WithRawTracer(config.Tracer),
		pubsub.WithDiscovery(config.Discovery()),
		pubsub.WithProtocolMatchFn(config.ProtoMatchFunc()),
		pubsub.WithGossipSubProtocols(config.Subprotocols()))
}

func (config overlayConfig) Context() context.Context {
	return config.CLI.Context
}

func (config overlayConfig) Namespace() string {
	return config.Vat.NS
}

func (config overlayConfig) Host() host.Host {
	return config.Vat.Host
}

func (config overlayConfig) Discovery() discovery.Discovery {
	// Dynamically dispatche advertisements and search queries to either:
	//
	// 1. the bootstrap service, iff the namespace matches the cluster topic; else
	// 2. the DHT-backed ambient peer discovery service.
	return boot.Namespace{
		Match: config.bootMatcher(),
		// Target:  config.PeX,  // TODO:  re-enable when PeX bugs are fixed
		Target:  config.Boot, // TODO:  remove when PeX bugs are fixed
		Default: routing.NewRoutingDiscovery(config.DHT),
	}
}

func (config overlayConfig) bootMatcher() func(string) bool {
	bootTopic := "floodsub:" + config.Namespace()
	return func(ns string) bool {
		return ns == bootTopic
	}
}

func (config overlayConfig) Proto() protocol.ID {
	// FIXME: For security, the cluster topic should not be present
	//        in the root pubsub capability server.

	//        The cluster topic should instead be provided as an
	//        entirely separate capability, negoaiated outside of
	//        the PubSub cap.

	// /casm/<casm-version>/ww/<version>/<ns>/meshsub/1.1.0
	return protoutil.Join(
		ww.Subprotocol(config.Namespace()),
		pubsub.GossipSubID_v11)
}

func (config overlayConfig) Matcher() protoutil.MatchFunc {
	proto, version := protoutil.Split(pubsub.GossipSubID_v11)
	return protoutil.Match(
		ww.NewMatcher(config.Namespace()),
		protoutil.Exactly(string(proto)),
		protoutil.SemVer(string(version)))
}

func (config overlayConfig) ProtoMatchFunc() pubsub.ProtocolMatchFn {
	match := config.Matcher()

	return func(local string) func(string) bool {
		if match.Match(local) {
			return match.Match
		}

		panic(fmt.Sprintf("match failed for local protocol %s", local))
	}
}

func (config overlayConfig) Features() func(pubsub.GossipSubFeature, protocol.ID) bool {
	supportGossip := config.Matcher()

	_, version := protoutil.Split(config.Proto())
	supportsPX := protoutil.Suffix(version)

	return func(feat pubsub.GossipSubFeature, proto protocol.ID) bool {
		switch feat {
		case pubsub.GossipSubFeatureMesh:
			return supportGossip.MatchProto(proto)

		case pubsub.GossipSubFeaturePX:
			return supportsPX.MatchProto(proto)

		default:
			return false
		}
	}
}

func (config overlayConfig) Subprotocols() ([]protocol.ID, func(pubsub.GossipSubFeature, protocol.ID) bool) {
	return []protocol.ID{config.Proto()}, config.Features()
}

func randomIdentityEd25519() (crypto.PrivKey, error) {
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
	return priv, err
}
