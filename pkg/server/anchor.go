package server

import (
	log "github.com/lthibault/log/pkg"
	"github.com/pkg/errors"

	capnp "zombiezen.com/go/capnproto2"

	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"

	"github.com/lthibault/wetware/internal/api"
	ww "github.com/lthibault/wetware/pkg"
	"github.com/lthibault/wetware/pkg/internal/routing"
	anchorpath "github.com/lthibault/wetware/pkg/util/anchor/path"
)

/*
	api.go contains the capnp api that is served by the host
*/

type rootAnchor struct {
	id  peer.ID
	log log.Logger
	routing.Table
	anchor
}

func newRootAnchor(log log.Logger, host host.Host, t routing.Table) rootAnchor {
	return rootAnchor{
		id:    host.ID(),
		log:   log,
		Table: t,
		anchor: anchor{
			log:  log,
			root: newAnchorTree(),
		},
	}
}

func (r rootAnchor) Protocol() protocol.ID {
	return ww.Protocol
}

func (r rootAnchor) Log() log.Logger {
	return r.log
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
		r.log.WithFields(log.F{
			"error":  err,
			"method": method,
		}).Error("rpc call failed")
	}

	return errors.Wrap(err, "internal server error")
}

type anchor struct {
	log  log.Logger
	root anchorNode
}

func (a anchor) Log() log.Logger {
	return a.log
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
		log:  a.log,
		root: node,
	})
}

func (a anchor) annotateErr(method string, err error) error {
	if err != nil {
		a.log.WithFields(log.F{
			"error":  err,
			"method": method,
			"path":   a.root.Path(),
		}).Error("rpc call failed")
	}

	return errors.Wrap(err, "internal server error")
}
