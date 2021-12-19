// Package client exports the Wetware client API.
package client

import (
	"context"

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

func (n Node) Close() error {
	return multierr.Combine(
		n.overlay.Close(), // MUST happen before n.Host.Close()
		n.host.Close())
}

// Bootstrap allows callers to hint to the routing system to get into a
// Boostrapped state and remain there. It is not a synchronous call.
//
// Bootstrap has no effect if routing is not configured.
func (n Node) Bootstrap(ctx context.Context) (err error) {
	if n.routing != nil {
		err = n.routing.Bootstrap(ctx)
	}

	return
}

func (n Node) Routing() routing.Routing { return n.routing }

func (n Node) PubSub() PubSub { return n.overlay }
