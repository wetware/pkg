package server

import (
	"github.com/pkg/errors"
	capnp "zombiezen.com/go/capnproto2"

	log "github.com/lthibault/log/pkg"

	"github.com/lthibault/wetware/internal/api"
	anchorpath "github.com/lthibault/wetware/pkg/util/anchor/path"
)

type anchor struct {
	log  log.Logger
	root anchorNode
}

func newAnchor(log log.Logger, root anchorNode) api.Anchor {
	return api.Anchor_ServerToClient(anchor{
		log:  log,
		root: root,
	})
}

func (a anchor) annotateErr(method string, err error) error {
	if err != nil {
		a.log.WithError(err).WithField("method", method).
			Error("rpc call failed")
	}

	return errors.Wrap(err, "internal server error")
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

		if err = item.SetAnchor(newAnchor(
			a.log.WithField("path", child.Node.Path()),
			child.Node,
		)); err != nil {
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

	node := a.root.Walk(anchorpath.Parts(path))
	return a.annotateErr("ls", call.Results.SetAnchor(newAnchor(
		a.log.WithField("path", node.Path()),
		node,
	)))
}
