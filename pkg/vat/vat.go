// Package vat provides a network abstraction for Cap'n Proto.
package vat

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/multiformats/go-multistream"
	ww "github.com/wetware/ww/pkg"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
)

var ErrInvalidNS = errors.New("invalid namespace")

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

type MetricReporter interface {
	CountAdd(key string, value int)
	CountSet(key string, value int)
	GaugeAdd(key string, value int)
	GaugeSet(key string, value int)
}

// Network wraps a libp2p Host and provides a high-level interface to
// a capability-oriented network.
type Network struct {
	NS      string
	Host    host.Host
	Metrics MetricReporter
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
		if err != multistream.ErrNotSupported {
			return nil, err
		}

		if n.isInvalidNS(vat.ID, c) {
			return nil, ErrInvalidNS
		}

		return nil, err // TODO:  catch multistream.ErrNotSupported
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
			n.gaugeMetrics(fmt.Sprintf("rpc.%s.open", id), 1)
			n.countMetrics("rpc.connect", 1)

			defer conn.Close()
			defer n.gaugeMetrics(fmt.Sprintf("rpc.%s.open", id), -1)
			defer n.countMetrics("rpc.disconnect", 1)

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

func (n Network) isInvalidNS(id peer.ID, c Capability) bool {
	ps, err := n.Host.Peerstore().GetProtocols(id)
	if err != nil {
		return false
	}

	for _, proto := range ps {
		if match(c, proto) {
			// the remote peer supports the capability, so it
			// has to be a namespace mismatch.
			return true
		}
	}

	return false // not a ns issue; proto actually unsupported
}

func (n Network) gaugeMetrics(name string, value int) {
	if n.Metrics != nil {
		n.Metrics.GaugeAdd(name, value)
	}
}

func (n Network) countMetrics(name string, value int) {
	if n.Metrics != nil {
		n.Metrics.CountAdd(name, value)
	}
}

// match the protocol, ignoring namespace
func match(c Capability, proto string) bool {
	for _, p := range c.Protocols() {
		if strings.HasSuffix(proto, string(p)) {
			return true
		}
	}

	return false
}

func bootstrapper(c Capability) *capnp.Client {
	if b, ok := c.(Bootstrapper); ok {
		return b.Bootstrap()
	}

	return nil
}
