package server

import (
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/pkg/errors"
	capnp "zombiezen.com/go/capnproto2"

	log "github.com/lthibault/log/pkg"
	"github.com/lthibault/wetware/internal/api"
	ww "github.com/lthibault/wetware/pkg"
	anchorpath "github.com/lthibault/wetware/pkg/util/anchor/path"
)

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
