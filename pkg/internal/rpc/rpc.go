package rpc

import (
	log "github.com/lthibault/log/pkg"

	capnp "zombiezen.com/go/capnproto2"
	"zombiezen.com/go/capnproto2/rpc"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
)

// Capability is a network object.  Remote hosts can hold references to a capability and
// call its methods.
type Capability interface {
	// Log returns the capability's logger.
	// It is used to log network information
	// related to the underlying object.
	Log() log.Logger

	// Protocol returns the capability's protocol identifier.
	Protocol() protocol.ID

	// Client returns the capnproto RPC client that points to the capability's main
	// exported interface.
	Client() capnp.Client
}

// Export a capability over a libp2p stream.
func Export(cap Capability) network.StreamHandler {
	return func(s network.Stream) {
		defer s.Reset()

		// TODO:  write a stream transport that uses a packed encoder/decoder pair
		//
		//  Difficulty:  easy.
		// 	https: //github.com/capnproto/go-capnproto2/blob/v2.18.0/rpc/transport.go
		conn := rpc.NewConn(rpc.StreamTransport(s), rpc.MainInterface(cap.Client()))

		if err := handleConn(conn); err != nil {
			cap.Log().WithError(err).Debug("rpc conn aborted")
		}
	}
}

func handleConn(conn *rpc.Conn) error {
	err := conn.Wait() // always returns an error
	if err == rpc.ErrConnClosed {
		return nil
	}

	return err
}
