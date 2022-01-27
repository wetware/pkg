package server

import (
	"errors"
	"io"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/lthibault/log"
	"go.uber.org/multierr"

	protoutil "github.com/wetware/casm/pkg/util/proto"
	rpcutil "github.com/wetware/ww/internal/util/rpc"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/cap/anchor"
	pscap "github.com/wetware/ww/pkg/cap/pubsub"
)

type capSet struct {
	cq chan struct{}

	Anchor anchor.Factory
	PubSub pscap.Factory
}

func newCapSet(a anchor.Factory, ps pscap.Factory) capSet {
	return capSet{
		cq:     make(chan struct{}),
		Anchor: a,
		PubSub: ps,
	}
}

func (cs capSet) Close() error {
	select {
	case <-cs.cq:
		return errors.New("already closed")

	default:
		close(cs.cq)
		return multierr.Combine(
			cs.Anchor.Close(),
			cs.PubSub.Close())
	}
}

// String returns the namespace containing the capability set.
func (cs capSet) String() string {
	return cs.Anchor.String()
}

func (cs capSet) registerRPC(h host.Host, log log.Logger) {
	var (
		match       = ww.NewMatcher(cs.String())
		matchPacked = match.Then(protoutil.Exactly("packed"))
	)

	h.SetStreamHandlerMatch(
		ww.Subprotocol(cs.String()),
		match,
		cs.newHandler(log, rpc.NewStreamTransport))

	h.SetStreamHandlerMatch(
		ww.Subprotocol(cs.String(), "packed"),
		matchPacked,
		cs.newHandler(log, rpc.NewPackedStreamTransport))
}

func (cs capSet) unregisterRPC(h host.Host) {
	h.RemoveStreamHandler(ww.Subprotocol(cs.String()))
	h.RemoveStreamHandler(ww.Subprotocol(cs.String(), "packed"))
}

func (cs capSet) newHandler(log log.Logger, f transportFactory) network.StreamHandler {
	return func(s network.Stream) {
		slog := log.With(streamFields(s))
		defer s.Close()

		conn := rpc.NewConn(f.NewTransport(s), &rpc.Options{
			BootstrapClient: cs.PubSub.New(nil).Client, // TODO:  AuthNegotiator or somesuch
			ErrorReporter: rpcutil.ErrReporterFunc(func(err error) {
				slog.Debug(err)
			}),
		})
		defer conn.Close()

		select {
		case <-conn.Done():
			slog.Debug("client hung up")

		case <-cs.cq:
			slog.Debug("shutting down")
		}
	}
}

type transportFactory func(io.ReadWriteCloser) rpc.Transport

func (f transportFactory) NewTransport(rwc io.ReadWriteCloser) rpc.Transport { return f(rwc) }

func streamFields(s network.Stream) log.F {
	return log.F{
		"peer":   s.Conn().RemotePeer(),
		"conn":   s.Conn().ID(),
		"proto":  s.Protocol(),
		"stream": s.ID(),
	}
}
