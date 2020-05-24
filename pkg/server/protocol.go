package server

import (
	log "github.com/lthibault/log/pkg"
	"github.com/pkg/errors"

	capnp "zombiezen.com/go/capnproto2"
	"zombiezen.com/go/capnproto2/rpc"

	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"

	"github.com/lthibault/wetware/internal/api"
	ww "github.com/lthibault/wetware/pkg"
	"github.com/lthibault/wetware/pkg/cluster"
)

/*
	api.go contains the capnp api that is served by the host
*/

func registerProtocols(f logFactory, host host.Host, r cluster.RoutingTable) {
	host.SetStreamHandler(ww.ClusterProtocol, clusterHandler(f, r))
	host.SetStreamHandler(ww.AnchorProtocol, hostAnchorHandler(f))
}

func clusterHandler(f logFactory, r cluster.RoutingTable) network.StreamHandler {
	return wrapHandler(f, handlerFunc(func(s stream) {
		defer s.Close()

		msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			s.Log().WithError(err).WithField("capnp_type", "message").
				Error("arena allocation failed")
			return
		}

		ps, err := api.NewRootPeerSet(seg)
		if err != nil {
			s.Log().WithError(err).WithField("capnp_type", "peerset").
				Error("arena allocation failed")
			return
		}

		peers := r.Peers()

		ids, err := ps.NewIds(int32(len(peers)))
		if err != nil {
			s.Log().WithError(err).WithField("capnp_type", "id list").
				Error("arena allocation failed")
			return
		}

		for i, p := range peers {
			if err = ids.Set(i, p.String()); err != nil {
				s.Log().WithError(err).WithField("capnp_type", "id list").
					Error("failed to set peer ID")
				return
			}
		}

		if err = capnp.NewPackedEncoder(s).Encode(msg); err != nil {
			s.Log().WithError(err).Error("failed to encode")
		}
	}))
}

func hostAnchorHandler(f logFactory) network.StreamHandler {

	root := newAnchorTree()

	return wrapHandler(f, handlerFunc(func(s stream) {
		defer s.Reset()

		a := newAnchor(f.Log(), root)
		conn := rpc.NewConn(streamTransport(s), rpc.MainInterface(a.Client))

		if err := waitRPC(conn); err != nil {
			s.Log().WithError(err).Debug("rpc conn aborted")
		}
	}))
}

func waitRPC(conn *rpc.Conn) error {
	switch err := conn.Wait(); err {
	case rpc.ErrConnClosed:
	default:
		if exc, ok := err.(rpc.Abort); ok {
			if exc.HasReason() {
				msg, err := exc.Reason()
				if err != nil {
					return errors.Wrap(err, "unable to decode reason")
				}

				return errors.New(msg)
			}

			return errors.New("remote terminated connection without reason")
		}

		return err
	}

	return nil
}

func streamTransport(s network.Stream) rpc.Transport {
	// TODO:  write a stream transport that uses a packed encoder/decoder pair
	//
	//  Difficulty:  easy.
	// 	https: //github.com/capnproto/go-capnproto2/blob/v2.18.0/rpc/transport.go
	return rpc.StreamTransport(s)
}

type stream interface {
	logFactory
	network.Stream
}

type handler interface {
	ServeP2P(stream)
}

type handlerFunc func(stream)

func (f handlerFunc) ServeP2P(s stream) {
	f(s)
}

func wrapHandler(f logFactory, h handler) network.StreamHandler {
	return func(s network.Stream) {
		h.ServeP2P(struct {
			logFactory
			network.Stream
		}{streamLogFactory(f, s), s})
	}
}

func streamLogFactory(f logFactory, s network.Stream) logFactory {
	return cachedLogFactory(func() log.Logger {
		return f.Log().WithField("proto", s.Protocol())
	})
}
