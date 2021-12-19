// Package client exports the Wetware client API.
package client

import (
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ctxutil "github.com/lthibault/util/ctx"
	"go.uber.org/multierr"
)

func DefaultRouting(h host.Host) (routing.Routing, error) {
	ctx := ctxutil.C(h.Network().Process().Closing())
	return dual.New(ctx, h,
		dual.DHTOption(dht.Mode(dht.ModeClient)))
}

type PubSub interface {
	Join(topic string, opt ...pubsub.TopicOpt) (*pubsub.Topic, error)
	Subscribe(topic string, opts ...pubsub.SubOpt) (*pubsub.Subscription, error)
	GetTopics() []string
	ListPeers(topic string) []peer.ID
}

type Node struct {
	host    host.Host
	routing routing.Routing
	overlay overlay
}

// String returns the cluster namespace
func (n Node) String() string {
	return n.overlay.String()
}

func (n Node) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"ns":        n.String(),
		"id":        n.host.ID(),
		"connected": !n.overlay.Orphaned(),
	}
}

func (n Node) Close() error {
	return multierr.Combine(
		n.overlay.Close(), // MUST happen before n.Host.Close()
		n.host.Close())
}

func (n Node) Routing() routing.Routing { return n.routing }

func (n Node) PubSub() PubSub { return n.overlay }

// GetClusterSubscription returns a subscription to the cluster topic.
// This topic is read-only for clients.  Users MUST close the
// subscription prior to calling n.Close().
func (n Node) GetClusterSubscription() (*pubsub.Subscription, error) {
	return n.overlay.t.Subscribe()
}
