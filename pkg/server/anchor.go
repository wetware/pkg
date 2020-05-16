package server

import (
	"github.com/pkg/errors"

	capnp "zombiezen.com/go/capnproto2"
	"zombiezen.com/go/capnproto2/rpc"

	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	log "github.com/lthibault/log/pkg"

	"github.com/lthibault/wetware/internal/api"
	ww "github.com/lthibault/wetware/pkg"
	"github.com/lthibault/wetware/pkg/cluster"
)

/*
	api.go contains the capnp api that is served by the host
*/

func registerProtocols(lp logProvider, host host.Host, r cluster.RoutingTable) {
	host.SetStreamHandler(ww.ClusterProtocol, clusterHandler(lp, r))
	host.SetStreamHandler(ww.AnchorProtocol, hostAnchorHandler(lp, host, r))
}

func clusterHandler(lp logProvider, r cluster.RoutingTable) network.StreamHandler {
	return wrapHandler(lp, handlerFunc(func(s stream) {
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

		s.Log().Info(ids)

		if err = capnp.NewPackedEncoder(s).Encode(msg); err != nil {
			s.Log().WithError(err).Error("failed to encode")
		}
	}))
}

func hostAnchorHandler(lp logProvider, host host.Host, r cluster.RoutingTable) network.StreamHandler {
	return wrapHandler(lp, hostAnchor{host: host, cluster: r})
}

type hostAnchor struct {
	host    host.Host
	cluster cluster.RoutingTable
}

func (a hostAnchor) ServeP2P(s stream) {
	defer s.Reset()

	export := api.Anchor_ServerToClient(a)

	conn := rpc.NewConn(streamTransport(s), rpc.MainInterface(export.Client))
	if err := waitRPC(conn); err != nil {
		s.Log().WithError(err).Debug("rpc conn aborted")
	}
}

func (a hostAnchor) Ls(call api.Anchor_ls) error {
	return errors.New("NOT IMPLEMENTED")
}

func (a hostAnchor) Walk(call api.Anchor_walk) error {
	return errors.New("NOT IMPLEMENTED")
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
	Log() log.Logger
	network.Stream
}

type handler interface {
	ServeP2P(stream)
}

type handlerFunc func(stream)

func (f handlerFunc) ServeP2P(s stream) {
	f(s)
}

func wrapHandler(lp logProvider, h handler) network.StreamHandler {
	return func(s network.Stream) {
		h.ServeP2P(struct {
			logProvider
			network.Stream
		}{streamLogProvider(lp, s), s})
	}
}

func streamLogProvider(lp logProvider, s network.Stream) logProvider {
	return logProviderFunc(func() log.Logger {
		return lp.Log().WithField("proto", s.Protocol())
	})
}
