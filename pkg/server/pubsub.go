package server

import (
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	discovery "github.com/libp2p/go-libp2p-discovery"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ctxutil "github.com/lthibault/util/ctx"
)

type PubSub interface {
	Join(topic string, opt ...pubsub.TopicOpt) (*pubsub.Topic, error)
	Subscribe(topic string, opts ...pubsub.SubOpt) (*pubsub.Subscription, error)
	GetTopics() []string
	ListPeers(topic string) []peer.ID
	BlacklistPeer(pid peer.ID)
	RegisterTopicValidator(topic string, val interface{}, opts ...pubsub.ValidatorOpt) error
	UnregisterTopicValidator(topic string) error
}

type PubSubFactory interface {
	New(host.Host, routing.ContentRouting) (PubSub, error)
}

type GossipsubFactory struct{}

func (GossipsubFactory) New(h host.Host, r routing.ContentRouting) (PubSub, error) {
	ctx := ctxutil.C(h.Network().Process().Closing())
	return pubsub.NewGossipSub(ctx, h,
		pubsub.WithDiscovery(discovery.NewRoutingDiscovery(r)))
}
