package vat

import (
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p-core/protocol"
	protoutil "github.com/wetware/casm/pkg/util/proto"
)

var packed = protoutil.Exactly("packed")

// BasicCap is a basic provider of Capability.  Most implementations
// will benefit from using this directly.  See pkg/cap/pubsub for an
// example.
type BasicCap []protocol.ID

func (c BasicCap) Protocols() []protocol.ID { return c }

func (c BasicCap) Upgrade(s Stream) rpc.Transport {
	if packed.MatchProto(s.Protocol()) {
		return rpc.NewPackedStreamTransport(s)
	}

	return rpc.NewStreamTransport(s)
}
