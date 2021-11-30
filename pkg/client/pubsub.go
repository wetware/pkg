package client

import (
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	discovery "github.com/libp2p/go-libp2p-discovery"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ctxutil "github.com/lthibault/util/ctx"
)

type PubSub interface {
	Join(topic string, opt ...pubsub.TopicOpt) (*pubsub.Topic, error)
	Subscribe(topic string, opts ...pubsub.SubOpt) (*pubsub.Subscription, error)
	GetTopics() []string
}

type PubSubFactory interface {
	New(host.Host, routing.ContentRouting) (PubSub, error)
}

type defaultPubSubFactory struct{}

func (defaultPubSubFactory) New(h host.Host, r routing.ContentRouting) (PubSub, error) {
	ctx := ctxutil.FromChan(h.Network().Process().Closing())
	return pubsub.NewGossipSub(ctx, h,
		pubsub.WithDiscovery(discovery.NewRoutingDiscovery(r)))
}
