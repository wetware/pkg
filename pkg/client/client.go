// Package client exports the Wetware client API.
package client

import (
	"context"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pscap "github.com/wetware/ww/pkg/cap/pubsub"
	"github.com/wetware/ww/pkg/vat"
)

type PubSub interface {
	Join(topic string, opt ...pubsub.TopicOpt) (*pubsub.Topic, error)
	Subscribe(topic string, opts ...pubsub.SubOpt) (*pubsub.Subscription, error)
	GetTopics() []string
	ListPeers(topic string) []peer.ID
}

type Node struct {
	vat  vat.Network
	conn *rpc.Conn
	ps   pscap.PubSub // conn's bootstrap capability
}

// String returns the cluster namespace
func (n Node) String() string { return n.vat.NS }

func (n Node) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"ns": n.String(),
		"id": n.vat.Host.ID(),
	}
}

// Bootstrap blocks until the context expires, or the
// node's capabilities resolve.  It is safe to cancel
// the context passed to Dial after this method returns.
func (n Node) Bootstrap(ctx context.Context) error {
	// TODO:  update this when we replace 'ps' with a
	//        capability set.
	return n.ps.Client.Resolve(ctx)
}

// Done returns a read-only channel that is closed when
// 'n' becomes disconnected from the cluster.
func (n Node) Done() <-chan struct{} {
	return n.conn.Done()
}

func (n Node) Close() error {
	n.ps.Release()

	return n.conn.Close()
}

func (n Node) PubSub() pscap.PubSub { return n.ps }
