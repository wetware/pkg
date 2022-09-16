package anchor

import (
	"context"

	"capnproto.org/go/capnp/v3"
	casm "github.com/wetware/casm/pkg"
	api "github.com/wetware/ww/internal/api/anchor"
)

// type Anchor interface {
// 	String() string
// 	Path() Path

// 	Ls(context.Context) (Iterator, capnp.ReleaseFunc)
// 	Walk(context.Context, string) Anchor

// 	Client() capnp.Client
// 	AddRef() Anchor
// 	Release()
// }

type Anchor api.Anchor

func (a Anchor) AddRef() Anchor {
	return Anchor(api.Anchor(a).AddRef())
}

func (a Anchor) Release() {
	capnp.Client(a).Release()
}

func (a Anchor) Ls(ctx context.Context) (Iterator, capnp.ReleaseFunc) {
	f, release := api.Anchor(a).Ls(ctx, nil)
	h := &handler{Future: casm.Future(f)}
	return Iterator{
		Seq:    h,
		Future: h,
	}, release
}

// Walk to the register located at path.
func (a Anchor) Walk(ctx context.Context, path string) (Anchor, capnp.ReleaseFunc) {
	p := NewPath(path)

	if p.IsRoot() {
		return Anchor(a), a.AddRef().Release
	}

	f, release := api.Anchor(a).Walk(ctx, walkParam(p))
	return Anchor(f.Anchor()), release
}

type Iterator casm.Iterator[Anchor]

type handler struct {
	casm.Future
	err error
	pos int
}

func (h *handler) Err() error {
	if h.err == nil {
		select {
		case <-h.Future.Done():
			_, h.err = h.Struct()
		default:
		}
	}

	return h.err
}

func (h *handler) Next() (a Anchor, ok bool) {
	if a, ok = h.anchor(h.pos); ok {
		h.pos++
	}

	return
}

func (h *handler) anchor(i int) (Anchor, bool) {
	res, ok := h.results()
	if !ok {
		return Anchor{}, false
	}

	children, err := res.Children()
	if err != nil {
		h.err = err
		return Anchor{}, false
	}

	a, err := children.At(i)
	if err != nil {
		h.err = err
		return Anchor{}, false
	}

	return Anchor(a), true
}

func (h *handler) results() (res api.Anchor_ls_Results, ok bool) {
	if err := h.Err(); err == nil {
		r, err := h.Struct()
		res = api.Anchor_ls_Results(r)
		ok = err == nil && res.HasChildren() && res.HasNames()
	}

	return
}

func walkParam(path Path) func(api.Anchor_walk_Params) error {
	return func(ps api.Anchor_walk_Params) error {
		return path.Bind(ps)
	}
}
