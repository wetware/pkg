package rpc

import (
	"context"
	"errors"
	"io"

	capnp "zombiezen.com/go/capnproto2"
	"zombiezen.com/go/capnproto2/rpc"

	"github.com/libp2p/go-libp2p-core/mux"
	"github.com/libp2p/go-libp2p-core/protocol"
	ww "github.com/wetware/ww/pkg"
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
	Client() *capnp.Client
}

// Handle an incoming stream with the supplied capability.
func Handle(ctx context.Context, log ww.Logger, cap Capability, rwc io.ReadWriteCloser) error {
	//
	// TODO(performance):  transport using packed encoding
	//
	conn := rpc.NewConn(rpc.NewStreamTransport(rwc), rpcOpts(log, cap))

	select {
	case <-conn.Done():
		return nil
	case <-ctx.Done():
		return conn.Close()
	}
}

func rpcOpts(log ww.Logger, cap Capability) *rpc.Options {
	return &rpc.Options{
		ErrorReporter:   errReporter{log.With(cap)},
		BootstrapClient: cap.Client(),
	}
}

type errReporter struct{ ww.Logger }

func (r errReporter) ReportError(err error) {
	if errors.Is(err, mux.ErrReset) {
		r.Debug("stream reset")
		return
	}

	// TODO:  capnp seems to flatten the error chain by formatting with %s.
	//		  Fix this upstream, then uncomment.

	// r.WithError(err).Error("error receiving message from remote vat")
	r.WithError(err).Debug("stream error")
}
