package server

import (
	"fmt"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/wetware/pkg/util/proto"
)

func (config Config) protoMatchFunc() pubsub.ProtocolMatchFn {
	match := matcher(config.NS)

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

func (config Config) subProtos() ([]protocol.ID, func(pubsub.GossipSubFeature, protocol.ID) bool) {
	return []protocol.ID{protoID(config.NS)}, config.features()
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

func (config Config) features() func(pubsub.GossipSubFeature, protocol.ID) bool {
	supportGossip := matcher(config.NS)

	_, version := proto.Split(protoID(config.NS))
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
