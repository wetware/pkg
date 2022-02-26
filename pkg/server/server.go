// package server exports the Wetware worker node.
package server

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/wetware/ww/pkg/vat"
)

type Node struct {
	id  uuid.UUID // instance ID
	vat vat.Network
	c   io.Closer
}

func New(ctx context.Context, vat vat.Network, ps PubSub, opt ...Option) (*Node, error) {
	return NewJoiner(opt...).Join(ctx, vat, ps)
}

func (n *Node) Close() error {
	return n.c.Close()
}

// String returns the cluster namespace
func (n *Node) String() string { return n.vat.NS }

func (n *Node) Loggable() map[string]interface{} {
	m := n.vat.Loggable()
	m["instance"] = n.id
	return m
}
