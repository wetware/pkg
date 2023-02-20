package runtime

import (
	"context"
	"fmt"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/lthibault/log"
	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/util/metrics"
	protoutil "github.com/wetware/casm/pkg/util/proto"
	ww "github.com/wetware/ww/pkg"
	ww_pubsub "github.com/wetware/ww/pkg/pubsub"
	"go.uber.org/fx"
)

type pubSubConfig struct {
	fx.In

	Ctx     context.Context
	Log     log.Logger
	Metrics metrics.Client
	Flag    Flags
	Vat     casm.Vat
	Boot    discovery.Discovery
}

func (c *Config) PubSub() fx.Option {
	return fx.Module("pubsub",
		fx.Provide(c.newPubSub),
		fx.Decorate(c.withPubSubDiscovery))
}

func (c *Config) newPubSub(config pubSubConfig) (*pubsub.PubSub, error) {
	return pubsub.NewGossipSub(config.Ctx, config.Vat.Host,
		pubsub.WithPeerExchange(true),
		pubsub.WithRawTracer(config.tracer()),
		pubsub.WithDiscovery(config.Boot),
		pubsub.WithProtocolMatchFn(config.protoMatchFunc()),
		pubsub.WithGossipSubProtocols(config.subProtos()),
		pubsub.WithPeerOutboundQueueSize(256),
	)
}

func (config pubSubConfig) tracer() ww_pubsub.Tracer {
	return ww_pubsub.Tracer{
		Log:     config.Log,
		Metrics: config.Metrics.WithPrefix("pubsub"),
	}
}

func (config pubSubConfig) protoMatchFunc() pubsub.ProtocolMatchFn {
	match := matcher(config.Flag)

	return func(local protocol.ID) func(protocol.ID) bool {
		if match.Match(local) {
			return match.Match
		}

		panic(fmt.Sprintf("match failed for local protocol %s", local))
	}
}

func (config pubSubConfig) features() func(pubsub.GossipSubFeature, protocol.ID) bool {
	supportGossip := matcher(config.Flag)

	_, version := protoutil.Split(protoID(config.Flag))
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

func matcher(flag Flags) protoutil.MatchFunc {
	proto, version := protoutil.Split(pubsub.GossipSubID_v11)
	return protoutil.Match(
		ww.NewMatcher(flag.String("ns")),
		protoutil.Exactly(string(proto)),
		protoutil.SemVer(string(version)))
}

func (config pubSubConfig) subProtos() ([]protocol.ID, func(pubsub.GossipSubFeature, protocol.ID) bool) {
	return []protocol.ID{protoID(config.Flag)}, config.features()
}

func protoID(flag Flags) protocol.ID {
	// FIXME: For security, the cluster topic should not be present
	//        in the root pubsub capability server.

	//        The cluster topic should instead be provided as an
	//        entirely separate capability, negoaiated outside of
	//        the PubSub cap.

	// /casm/<casm-version>/ww/<version>/<ns>/meshsub/1.1.0
	return protoutil.Join(
		ww.Subprotocol(flag.String("ns")),
		pubsub.GossipSubID_v11)
}
