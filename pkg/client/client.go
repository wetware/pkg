// Package client exports the Wetware client API.
package client

import (
	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p-core/routing"
)

type Conn interface {
	// Done returns a channel that is closed when the underlying RPC transport
	// is closed.  Callers MUST subsequently call Close() in order to clean up
	// shared resources. Callers MAY call Close() before the channel is closed.
	Done() <-chan struct{}

	// Close the underlying transport and libp2p host services.  Callers MUST
	// call Close() when they are finished with the client.
	Close() error
}

type Node struct {
	Conn
	c *capnp.Client

	Routing routing.Routing
	PubSub  PubSub
}

// Object returns an RPC client that references the
// underlying object.
func (n Node) Object() *capnp.Client { return n.c }
