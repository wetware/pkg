// Package vat provides a network abstraction for Cap'n Proto.
package vat

import (
	"context"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	ww "github.com/wetware/ww/pkg"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
)

type Capability interface {
	// Protocols returns the IDs for the given capability.
	// Implementations SHOULD order protocol identifiers in decreasing
	// order of priority.
	Protocols() []protocol.ID

	// Upgrade a raw byte-stream to an RPC transport.  Implementations
	// MAY select a Transport impmlementation based on the protocol ID
	// returned by 'Stream.Protocol'.
	Upgrade(Stream) rpc.Transport
}

type Bootstrapper interface {
	Bootstrap() *capnp.Client
}

type Stream interface {
	Protocol() protocol.ID
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
}

type ClientProvider interface {
	// Client returns the client capability to be exported.  It is called
	// once for each incoming Stream, so implementations may either share
	// a single global object, or instantiate a new object for each call.
	Client() *capnp.Client
}

// Network wraps a libp2p Host and provides a high-level interface to
// a capability-oriented network.
type Network struct {
	NS   string
	Host host.Host
}

func (n Network) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"ns": n.NS,
		"id": n.Host.ID(),
	}
}

// Connect to a capability hostend on vat.  The context is used only
// when negotiating network connections and is safe to cancel when a
// call to 'Connect' returns. The RPC connection is returned without
// waiting for the remote capability to resolve.  Users MAY refer to
// the 'Bootstrap' method on 'rpc.Conn' to resolve the connection.
//
// The 'Addrs' field of 'vat' MAY be empty, in which case the network
// will will attempt to discover a valid address.
//
// If 'c' satisfies the 'Bootstrapper' interface, the client returned
// by 'c.Bootstrap()' is provided to the RPC connection as a bootstrap
// capability.
func (n Network) Connect(ctx context.Context, vat peer.AddrInfo, c Capability) (*rpc.Conn, error) {
	if len(vat.Addrs) > 0 {
		if err := n.Host.Connect(ctx, vat); err != nil {
			return nil, err
		}
	}

	s, err := n.Host.NewStream(ctx, vat.ID, n.protocolsFor(c)...)
	if err != nil {
		return nil, err
	}

	return rpc.NewConn(c.Upgrade(s), &rpc.Options{
		BootstrapClient: bootstrapper(c),
	}), nil
}

// Export a capability, making it available to other vats in the network.
func (n Network) Export(c Capability, boot ClientProvider) {
	for _, id := range n.protocolsFor(c) {
		n.Host.SetStreamHandler(id, func(s network.Stream) {
			defer s.Close()

			conn := rpc.NewConn(c.Upgrade(s), &rpc.Options{
				BootstrapClient: boot.Client(),
			})
			defer conn.Close()

			<-conn.Done()
		})
	}
}

// Embargo ceases to export 'c'.  New calls to 'Connect' are guaranteed
// to fail for 'c' after 'Embargo' returns. Existing RPC connections on
// 'c' are unaffected.
func (n Network) Embargo(c Capability) {
	for _, id := range n.protocolsFor(c) {
		n.Host.RemoveStreamHandler(id)
	}
}

func (n Network) protocolsFor(c Capability) []protocol.ID {
	ps := make([]protocol.ID, len(c.Protocols()))
	for i, id := range protocol.ConvertToStrings(c.Protocols()) {
		ps[i] = ww.Subprotocol(n.NS, id)
	}
	return ps
}

func bootstrapper(c Capability) *capnp.Client {
	if b, ok := c.(Bootstrapper); ok {
		return b.Bootstrap()
	}

	return nil
}
