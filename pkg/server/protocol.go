package server

import (
	log "github.com/lthibault/log/pkg"
	"github.com/pkg/errors"

	capnp "zombiezen.com/go/capnproto2"
	"zombiezen.com/go/capnproto2/rpc"

	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"

	"github.com/lthibault/wetware/internal/api"
	ww "github.com/lthibault/wetware/pkg"
	"github.com/lthibault/wetware/pkg/cluster"
)

/*
	api.go contains the capnp api that is served by the host
*/

func registerProtocols(f logFactory, host host.Host, r cluster.RoutingTable) {
	for _, iface := range []remoteInterface{
		anchor{logFactory: f, root: newAnchorTree()},
		router{logFactory: f, RoutingTable: r},
	} {
		host.SetStreamHandler(iface.Proto(), export(iface))
	}

	// host.SetStreamHandler(ww.ClusterProtocol, clusterHandler(f, r))
	// host.SetStreamHandler(ww.AnchorProtocol, hostAnchorHandler(f))
}

type router struct {
	logFactory
	cluster.RoutingTable
}

func (router) Proto() protocol.ID {
	return ww.ClusterProtocol
}

func (r router) Export() capnp.Client {
	return api.Router_ServerToClient(r).Client
}

func (r router) Ls(call api.Router_ls) error {
	peers := r.Peers()

	view, err := call.Results.NewView(int32(len(peers)))
	if err != nil {
		return r.annotateErr("ls", err)
	}

	for i, p := range peers {
		if err = view.Set(i, p.String()); err != nil {
			return r.annotateErr("ls", err)
		}
	}

	return nil
}

func (r router) annotateErr(method string, err error) error {
	if err != nil {
		r.Log().WithFields(log.F{
			"error":  err,
			"method": method,
			"proto":  r.Proto(),
		}).Error("rpc call failed")
	}

	return errors.Wrap(err, "internal server error")
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

func export(iface remoteInterface) network.StreamHandler {
	return func(s network.Stream) {
		defer s.Reset()

		conn := rpc.NewConn(streamTransport(s), rpc.MainInterface(iface.Export()))

		if err := waitRPC(conn); err != nil {
			iface.Log().WithError(err).Debug("rpc conn aborted")
		}
	}
}

type remoteInterface interface {
	Log() log.Logger
	Proto() protocol.ID
	Export() capnp.Client
}
