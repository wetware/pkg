// Package client exports the Wetware client API.
package client

import (
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pscap "github.com/wetware/ww/pkg/cap/pubsub"
)

type PubSub interface {
	Join(topic string, opt ...pubsub.TopicOpt) (*pubsub.Topic, error)
	Subscribe(topic string, opts ...pubsub.SubOpt) (*pubsub.Subscription, error)
	GetTopics() []string
	ListPeers(topic string) []peer.ID
}

type Node struct {
	ns string
	h  host.Host
	ps pscap.PubSub
}

// String returns the cluster namespace
func (n Node) String() string { return n.ns }

func (n Node) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"ns": n.String(),
		"id": n.h.ID(),
	}
}

func (n Node) Close() error {
	defer n.ps.Client.Release()

	return n.h.Close()
}

func (n Node) PubSub() pscap.PubSub { return n.ps }
