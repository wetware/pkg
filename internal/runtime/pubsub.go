package runtime

import (
	"fmt"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/protocol"
	casm "github.com/wetware/casm/pkg"
	protoutil "github.com/wetware/casm/pkg/util/proto"
	ww "github.com/wetware/ww/pkg"
	ww_pubsub "github.com/wetware/ww/pkg/pubsub"
	"go.uber.org/fx"
)

type pubSubConfig struct {
	fx.In

	Vat  casm.Vat
	Boot discovery.Discovery
}

func (c *Config) PubSub() fx.Option {
	return fx.Module("pubsub",
		fx.Provide(c.newPubSub),
		fx.Decorate(c.withPubSubDiscovery))
}

func (c *Config) newPubSub(env Env, config pubSubConfig) (*pubsub.PubSub, error) {
	return pubsub.NewGossipSub(env.Context(), config.Vat.Host,
		pubsub.WithPeerExchange(true),
		pubsub.WithRawTracer(config.tracer(env)),
		pubsub.WithDiscovery(config.Boot),
		pubsub.WithProtocolMatchFn(config.protoMatchFunc(env)),
		pubsub.WithGossipSubProtocols(config.subProtos(env)))
}

func (pubSubConfig) tracer(env Env) ww_pubsub.Tracer {
	return ww_pubsub.Tracer{
		Log:     env.Log(),
		Metrics: env.Metrics().WithPrefix("pubsub"),
	}
}

func (config pubSubConfig) protoMatchFunc(env Env) pubsub.ProtocolMatchFn {
	match := matcher(env)

	return func(local string) func(string) bool {
		if match.Match(local) {
			return match.Match
		}

		panic(fmt.Sprintf("match failed for local protocol %s", local))
	}
}

func (config pubSubConfig) features(env Env) func(pubsub.GossipSubFeature, protocol.ID) bool {
	supportGossip := matcher(env)

	_, version := protoutil.Split(protoID(env))
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

func matcher(env Env) protoutil.MatchFunc {
	proto, version := protoutil.Split(pubsub.GossipSubID_v11)
	return protoutil.Match(
		ww.NewMatcher(env.String("ns")),
		protoutil.Exactly(string(proto)),
		protoutil.SemVer(string(version)))
}

func (config pubSubConfig) subProtos(env Env) ([]protocol.ID, func(pubsub.GossipSubFeature, protocol.ID) bool) {
	return []protocol.ID{protoID(env)}, config.features(env)
}

func protoID(env Env) protocol.ID {
	// FIXME: For security, the cluster topic should not be present
	//        in the root pubsub capability server.

	//        The cluster topic should instead be provided as an
	//        entirely separate capability, negoaiated outside of
	//        the PubSub cap.

	// /casm/<casm-version>/ww/<version>/<ns>/meshsub/1.1.0
	return protoutil.Join(
		ww.Subprotocol(env.String("ns")),
		pubsub.GossipSubID_v11)
}
