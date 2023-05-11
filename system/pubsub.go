package system

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/lthibault/log"
	"go.uber.org/fx"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/boot"
	protoutil "github.com/wetware/casm/pkg/util/proto"
	ww "github.com/wetware/ww/pkg"
)

type PubSubConfig struct {
	fx.In

	NS   string
	Vat  casm.Vat
	DHT  *dual.DHT
	Boot discovery.Discovery
}

func PubSub(ctx context.Context, config PubSubConfig) (*pubsub.PubSub, error) {
	return pubsub.NewGossipSub(ctx, config.Vat.Host,
		pubsub.WithPeerExchange(true),
		pubsub.WithDiscovery(config.discovery()),
		pubsub.WithProtocolMatchFn(config.protoMatchFunc()),
		pubsub.WithGossipSubProtocols(config.subProtos()))
}

func (config PubSubConfig) discovery() *boot.Namespace {
	// Dynamically dispatch advertisements and queries to either:
	//
	//  1. the bootstrap service, iff namespace matches cluster topic; else
	//  2. the DHT-backed discovery service.
	bootTopic := "floodsub:" + config.NS
	match := func(ns string) bool {
		return ns == bootTopic
	}

	target := logMetricDisc{
		disc: config.Boot,
		advt: config.Boot,
		metrics: bootMetrics{
			Log: config.Vat.Logger,
		},
	}

	return &boot.Namespace{
		Match:   match,
		Target:  trimPrefixDisc{target},
		Default: routing.NewRoutingDiscovery(config.DHT),
	}
}

func (config PubSubConfig) protoMatchFunc() pubsub.ProtocolMatchFn {
	match := matcher(config.NS)

	return func(local protocol.ID) func(protocol.ID) bool {
		if match.Match(local) {
			return match.Match
		}

		panic(fmt.Sprintf("match failed for local protocol %s", local))
	}
}

func (config PubSubConfig) features() func(pubsub.GossipSubFeature, protocol.ID) bool {
	supportGossip := matcher(config.NS)

	_, version := protoutil.Split(protoID(config.NS))
	supportsPX := protoutil.Suffix(version)

	return func(feat pubsub.GossipSubFeature, proto protocol.ID) bool {
		switch feat {
		case pubsub.GossipSubFeatureMesh:
			return supportGossip.Match(proto)

		case pubsub.GossipSubFeaturePX:
			return supportsPX.Match(proto)

		default:
			return false
		}
	}
}

func matcher(ns string) protoutil.MatchFunc {
	proto, version := protoutil.Split(pubsub.GossipSubID_v11)
	return protoutil.Match(
		ww.NewMatcher(ns),
		protoutil.Exactly(string(proto)),
		protoutil.SemVer(string(version)))
}

func (config PubSubConfig) subProtos() ([]protocol.ID, func(pubsub.GossipSubFeature, protocol.ID) bool) {
	return []protocol.ID{protoID(config.NS)}, config.features()
}

func protoID(ns string) protocol.ID {
	// FIXME: For security, the cluster topic should not be present
	//        in the root pubsub capability server.

	//        The cluster topic should instead be provided as an
	//        entirely separate capability, negoaiated outside of
	//        the PubSub cap.

	// /casm/<casm-version>/ww/<version>/<ns>/meshsub/1.1.0
	return protoutil.Join(
		ww.Subprotocol(ns),
		pubsub.GossipSubID_v11)
}

// Trims the "floodsub:" prefix from the namespace.  This is needed because
// clients do not use pubsub, and will search for the exact namespace string.
type trimPrefixDisc struct{ discovery.Discovery }

func (b trimPrefixDisc) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	ns = strings.TrimPrefix(ns, "floodsub:")
	return b.Discovery.FindPeers(ctx, ns, opt...)
}

func (b trimPrefixDisc) Advertise(ctx context.Context, ns string, opt ...discovery.Option) (time.Duration, error) {
	ns = strings.TrimPrefix(ns, "floodsub:")
	return b.Discovery.Advertise(ctx, ns, opt...)
}

type logMetricDisc struct {
	metrics bootMetrics
	disc    discovery.Discoverer
	advt    discovery.Advertiser
}

func (b logMetricDisc) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	b.metrics.OnFindPeers(ns)
	return b.disc.FindPeers(ctx, ns, opt...)
}

func (b logMetricDisc) Advertise(ctx context.Context, ns string, opt ...discovery.Option) (time.Duration, error) {
	b.metrics.OnAdvertise(ns)
	return b.advt.Advertise(ctx, ns, opt...)
}

type bootMetrics struct {
	Log log.Logger
}

func (m bootMetrics) OnFindPeers(ns string) {
	m.Log.Debug("bootstrapping namespace")
}

func (m bootMetrics) OnAdvertise(ns string) {
	m.Log.Debug("advertising namespace")
}
