package anchor

import (
	"context"

	"github.com/libp2p/go-libp2p-core/protocol"
	capnp "zombiezen.com/go/capnproto2"

	"github.com/lthibault/wetware/internal/api"
	ww "github.com/lthibault/wetware/pkg"
	"github.com/lthibault/wetware/pkg/internal/rpc"
)

type ls struct {
	rpc.Terminal
	p api.Anchor_ls_Results_Promise
}

// Protocol ID
func (ls) Protocol() protocol.ID {
	return ww.Protocol
}

// HandleRPC for `ls`
func (h *ls) HandleRPC(ctx context.Context, c capnp.Client) {
	h.p = api.Anchor{Client: c}.
		Ls(ctx, func(api.Anchor_ls_Params) error { return nil })
}

// Resolve .
func (h *ls) Resolve() ([]ww.Anchor, error) {
	res, err := h.p.Struct()
	if err != nil {
		return nil, err
	}

	return parseLs(res, rootLsHandler{h.Terminal})
}

func parseLs(res api.Anchor_ls_Results, h lsHandler) ([]ww.Anchor, error) {
	cs, err := res.Children()
	if err != nil {
		return nil, err
	}

	as := make([]ww.Anchor, cs.Len())
	for i := range as {
		if as[i], err = h.Handle(cs.At(i)); err != nil {
			return as[:i-1], err
		}
	}

	return as, err
}

type lsHandler interface {
	Handle(api.Anchor_SubAnchor) (ww.Anchor, error)
}

type rootLsHandler struct{ rpc.Terminal }

func (h rootLsHandler) Handle(a api.Anchor_SubAnchor) (ww.Anchor, error) {
	path, err := a.Path()
	if err != nil {
		return nil, err
	}

	return hostAnchor{
		d: rpc.DialString(path),
		t: h.Terminal,
	}, nil
}

type anchorLsHandler []string

func (h anchorLsHandler) Handle(a api.Anchor_SubAnchor) (ww.Anchor, error) {
	subpath, err := a.Path()
	if err != nil {
		return nil, err
	}

	return anchor{
		path:           append(path(h), subpath),
		anchorProvider: a,
	}, nil
}
