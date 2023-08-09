package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/lthibault/log"

	"github.com/wetware/ww/boot"
	"github.com/wetware/ww/util/proto"
)

func (cfg Config) newPubSub(ctx context.Context, pc pubSubConfig) (*pubsub.PubSub, error) {
	return pubsub.NewGossipSub(ctx, pc.Host,
		pubsub.WithPeerExchange(true),
		// pubsub.WithRawTracer(cfg.tracer()),
		pubsub.WithDiscovery(pc.NewDiscovery(ctx)),
		pubsub.WithProtocolMatchFn(cfg.protoMatchFunc()),
		pubsub.WithGossipSubProtocols(cfg.subProtos()),
		pubsub.WithPeerOutboundQueueSize(1024),
		pubsub.WithValidateQueueSize(1024),
	)
}

func (cfg Config) protoMatchFunc() pubsub.ProtocolMatchFn {
	match := matcher(cfg.NS)

	return func(local protocol.ID) func(protocol.ID) bool {
		if match.Match(local) {
			return match.Match
		}

		panic(fmt.Sprintf("match failed for local protocol %s", local))
	}
}

func matcher(ns string) proto.MatchFunc {
	base, version := proto.Split(pubsub.GossipSubID_v11)
	return proto.Match(
		proto.NewMatcher(ns),
		proto.Exactly(string(base)),
		proto.SemVer(string(version)))
}

func (cfg Config) subProtos() ([]protocol.ID, func(pubsub.GossipSubFeature, protocol.ID) bool) {
	return []protocol.ID{protoID(cfg.NS)}, cfg.features()
}

func protoID(ns string) protocol.ID {
	// FIXME: For security, the cluster topic should not be present
	//        in the root pubsub capability server.

	//        The cluster topic should instead be provided as an
	//        entirely separate capability, negoaiated outside of
	//        the PubSub cap.

	// /ww/<version>/<ns>/meshsub/1.1.0
	return proto.Join(
		proto.Root(ns),
		pubsub.GossipSubID_v11)
}

func (cfg Config) features() func(pubsub.GossipSubFeature, protocol.ID) bool {
	supportGossip := matcher(cfg.NS)

	_, version := proto.Split(protoID(cfg.NS))
	supportsPX := proto.Suffix(version)

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

type pubSubConfig struct {
	Logger    log.Logger
	NS        string
	Host      host.Host
	Discovery discovery.Discovery
	DHT       *dual.DHT
}

func (cfg *pubSubConfig) NewDiscovery(ctx context.Context) *boot.Namespace {
	// Dynamically dispatch advertisements and queries to either:
	//
	//  1. the bootstrap service, iff namespace matches cluster topic; else
	//  2. the DHT-backed discovery service.
	bootTopic := "floodsub:" + cfg.NS
	match := func(ns string) bool {
		return ns == bootTopic
	}

	target := loggingDiscovery{
		Logger:    cfg.Logger,
		Discovery: cfg.Discovery,
	}

	return &boot.Namespace{
		Match:   match,
		Target:  trimPrefixDisc{target},
		Default: routing.NewRoutingDiscovery(cfg.DHT),
	}
}

type loggingDiscovery struct {
	Logger log.Logger
	discovery.Discovery
}

func (b loggingDiscovery) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	b.Logger.Debug("bootstrapping namespace")
	return b.Discovery.FindPeers(ctx, ns, opt...)
}

func (b loggingDiscovery) Advertise(ctx context.Context, ns string, opt ...discovery.Option) (time.Duration, error) {
	b.Logger.Debug("advertising namespace")
	return b.Discovery.Advertise(ctx, ns, opt...)
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
