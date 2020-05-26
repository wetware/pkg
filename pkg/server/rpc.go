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
	anchorpath "github.com/lthibault/wetware/pkg/util/anchor/path"
)

/*
	api.go contains the capnp api that is served by the host
*/

var (
	_ remoteInterface = (*router)(nil)
	_ remoteInterface = (*anchor)(nil)
)

func rpcRegisterAll(f logFactory, host host.Host, r cluster.RoutingTable) {
	for _, iface := range []remoteInterface{
		anchor{logFactory: f, root: newAnchorTree()},
		router{logFactory: f, RoutingTable: r},
	} {
		host.SetStreamHandler(iface.Proto(), export(iface))
	}
}

type router struct {
	logFactory
	cluster.RoutingTable
}

func (router) Proto() protocol.ID {
	return ww.RouterProtocol
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

type anchor struct {
	logFactory
	root anchorNode
}

func (anchor) Proto() protocol.ID {
	return ww.AnchorProtocol
}

func (a anchor) Export() capnp.Client {
	return api.Anchor_ServerToClient(a).Client
}

func (a anchor) Ls(call api.Anchor_ls) error {
	children := a.root.List()

	cs, err := call.Results.NewChildren(int32(len(children)))
	if err != nil {
		return a.annotateErr("ls", err)
	}

	for i, child := range a.root.List() {
		_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			break
		}

		item, err := api.NewRootAnchor_SubAnchor(seg)
		if err != nil {
			break
		}

		if err = item.SetPath(child.Path); err != nil {
			break
		}

		if err = item.SetAnchor(a.subAnchor(child.Node)); err != nil {
			break
		}

		if err = cs.Set(i, item); err != nil {
			break
		}
	}

	return a.annotateErr("ls", err)
}

func (a anchor) Walk(call api.Anchor_walk) error {
	path, err := call.Params.Path()
	if err != nil {
		return a.annotateErr("walk", err)
	}

	child := a.subAnchor(a.root.Walk(anchorpath.Parts(path)))
	return a.annotateErr("ls", call.Results.SetAnchor(child))
}

func (a anchor) subAnchor(node anchorNode) api.Anchor {
	return api.Anchor_ServerToClient(anchor{
		logFactory: a.logFactory,
		root:       node,
	})
}

func (a anchor) annotateErr(method string, err error) error {
	if err != nil {
		a.Log().WithFields(log.F{
			"error":  err,
			"method": method,
			"path":   a.root.Path(),
			"proto":  a.Proto(),
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
