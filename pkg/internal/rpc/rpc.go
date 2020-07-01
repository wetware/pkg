package rpc

import (
	"io"

	capnp "zombiezen.com/go/capnproto2"
	"zombiezen.com/go/capnproto2/rpc"

	"github.com/libp2p/go-libp2p-core/protocol"
)

// Capability is a network object.  Remote hosts can hold references to a capability and
// call its methods.
type Capability interface {
	// Log returns the capability's log fields.
	Loggable() map[string]interface{}

	// Protocol returns the capability's protocol identifier.
	Protocol() protocol.ID

	// Client returns the capnproto RPC client that points to the capability's main
	// exported interface.
	Client() capnp.Client
}

// Handle an incoming stream with the supplied capability.
func Handle(cap Capability, rwc io.ReadWriteCloser) error {
	// TODO:  write a stream transport that uses a packed encoder/decoder pair
	//
	//  Difficulty:  easy.
	// 	https: //github.com/capnproto/go-capnproto2/blob/v2.18.0/rpc/transport.go
	conn := rpc.NewConn(rpc.StreamTransport(rwc), rpc.MainInterface(cap.Client()))

	err := conn.Wait() // always returns an error
	if err == rpc.ErrConnClosed {
		return nil
	}

	return err
}
