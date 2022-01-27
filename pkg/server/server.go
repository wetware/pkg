// package server exports the Wetware worker node.
package server

import (
	"context"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/host"
)

type Node struct {
	id uuid.UUID // instance ID
	h  host.Host
	c  capSet
}

func New(ctx context.Context, h host.Host, ps PubSub, opt ...Option) (*Node, error) {
	return NewJoiner(opt...).Join(ctx, h, ps)
}

func (n *Node) Close() error {
	n.c.unregisterRPC(n.h)
	return n.c.Close()
}

// String returns the cluster namespace
func (n *Node) String() string { return n.c.String() }

func (n *Node) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"ns":       n.String(),
		"id":       n.h.ID(),
		"instance": n.id,
	}
}
