// Package client exports the Wetware client API.
package client

import (
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pscap "github.com/wetware/ww/pkg/cap/pubsub"
	"go.uber.org/multierr"
)

type PubSub interface {
	Join(topic string, opt ...pubsub.TopicOpt) (*pubsub.Topic, error)
	Subscribe(topic string, opts ...pubsub.SubOpt) (*pubsub.Subscription, error)
	GetTopics() []string
	ListPeers(topic string) []peer.ID
}

type Node struct {
	ns   string
	h    host.Host
	conn *rpc.Conn
	ps   pscap.PubSub // conn's bootstrap capability
}

// String returns the cluster namespace
func (n Node) String() string { return n.ns }

func (n Node) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"ns": n.String(),
		"id": n.h.ID(),
	}
}

// Done returns a read-only channel that receives when n becomes
// disconnected from the cluster.
func (n Node) Done() <-chan struct{} {
	return n.conn.Done()
}

func (n Node) Close() error {
	n.ps.Release() // belt-and-suspenders

	return multierr.Combine(
		n.conn.Close(),
		n.h.Close())
}

func (n Node) PubSub() pscap.PubSub { return n.ps }
