// Package client exports the Wetware client API.
package client

import (
	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	"go.uber.org/multierr"
)

type Node struct {
	h  host.Host
	r  routing.Routing
	ps PubSub

	conn *rpc.Conn
	cap  *capnp.Client
}

// Done returns a channel that is closed when the underlying RPC transport
// is closed.  Callers MUST subsequently call Close() in order to clean up
// shared resources. Callers MAY call Close() before the channel is closed.
func (n Node) Done() <-chan struct{} { return n.conn.Done() }

// Close the underlying transport and libp2p host services.  Callers MUST
// call Close() when they are finished with the client.
func (n Node) Close() error {
	// This probably isn't necessary given that we're going to terminate
	// the connection, but doing so is technically part of Bootstrap's
	// contract, and this ensures any outstanding futures are rejected
	// prior to disconnecting.
	defer n.cap.Release()

	return multierr.Append(
		n.conn.Close(), // Close the QUIC stream first...
		n.h.Close())    // ...then close the libp2p host.
}

// PubSub returns a pubsub capability.
func (n Node) PubSub() PubSub { return n.ps }
