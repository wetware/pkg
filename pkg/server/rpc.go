package server

import (
	log "github.com/lthibault/log/pkg"
	"github.com/pkg/errors"

	capnp "zombiezen.com/go/capnproto2"
	"zombiezen.com/go/capnproto2/rpc"

	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/lthibault/wetware/internal/api"
	ww "github.com/lthibault/wetware/pkg"
	"github.com/lthibault/wetware/pkg/cluster"
	anchorpath "github.com/lthibault/wetware/pkg/util/anchor/path"
)

/*
	api.go contains the capnp api that is served by the host
*/

var (
	_ capability = (*rootAnchor)(nil)
	_ capability = (*anchor)(nil)
)

func registerRPC(f logFactory, host host.Host, r cluster.RoutingTable) {
	host.SetStreamHandler(ww.Protocol, export(newRootAnchor(f, host, r)))
}

type rootAnchor struct {
	id peer.ID
	logFactory
	cluster.RoutingTable
	anchor
}

func newRootAnchor(f logFactory, host host.Host, r cluster.RoutingTable) rootAnchor {
	return rootAnchor{
		id:           host.ID(),
		logFactory:   f,
		RoutingTable: r,
		anchor: anchor{
			logFactory: f,
			root:       newAnchorTree(),
		},
	}
}

func (r rootAnchor) Client() capnp.Client {
	return api.Anchor_ServerToClient(r).Client
}

func (r rootAnchor) Ls(call api.Anchor_ls) error {
	peers := r.Peers()

	cs, err := call.Results.NewChildren(int32(len(peers)))
	if err != nil {
		return r.annotateErr("ls", err)
	}

	for i, p := range peers {
		_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			return r.annotateErr("ls", err)
		}

		a, err := api.NewAnchor_SubAnchor(seg)
		if err != nil {
			return r.annotateErr("ls", err)
		}

		a.SetRoot()

		if err = a.SetPath(p.String()); err != nil {
			return r.annotateErr("ls", err)
		}

		if err = cs.Set(i, a); err != nil {
			return r.annotateErr("ls", err)
		}
	}

	return nil
}

func (r rootAnchor) Walk(call api.Anchor_walk) error {
	path, err := call.Params.Path()
	if err != nil {
		return r.annotateErr("walk", err)
	}

	parts := anchorpath.Parts(path)
	if id := parts[0]; id != r.id.String() {
		return errors.Errorf("bad request: id mismatch (expected %s, got %s)", r.id, id)
	}

	// pop the path head before passing the call down to the `anchor`.
	if err = call.Params.SetPath(anchorpath.Join(parts[1:])); err != nil {
		return r.annotateErr("walk", err)
	}

	return r.anchor.Walk(call)
}

func (r rootAnchor) annotateErr(method string, err error) error {
	if err != nil {
		r.Log().WithFields(log.F{
			"error":  err,
			"method": method,
		}).Error("rpc call failed")
	}

	return errors.Wrap(err, "internal server error")
}

type anchor struct {
	logFactory
	root anchorNode
}

func (a anchor) Client() capnp.Client {
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

func export(c capability) network.StreamHandler {
	return func(s network.Stream) {
		defer s.Reset()

		// TODO:  write a stream transport that uses a packed encoder/decoder pair
		//
		//  Difficulty:  easy.
		// 	https: //github.com/capnproto/go-capnproto2/blob/v2.18.0/rpc/transport.go
		conn := rpc.NewConn(rpc.StreamTransport(s), rpc.MainInterface(c.Client()))

		if err := waitRPC(conn); err != nil {
			c.Log().WithError(err).Debug("rpc conn aborted")
		}
	}
}

type capability interface {
	Log() log.Logger
	Client() capnp.Client
}
