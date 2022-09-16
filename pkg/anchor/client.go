package anchor

import (
	"context"

	"capnproto.org/go/capnp/v3"
	casm "github.com/wetware/casm/pkg"
	api "github.com/wetware/ww/internal/api/anchor"
)

type Anchor interface {
	String() string
	Path() Path

	Ls(context.Context) (Iterator, capnp.ReleaseFunc)
	Walk(context.Context, string) Anchor

	Client() capnp.Client
	AddRef() Anchor
	Release()
}

// type client struct {
// 	path Path
// 	api.Anchor
// }

type Client struct {
	api.Anchor
	path Path
}

func (a Client) AddRef() Client {
	return Client(api.Anchor(a).AddRef())
}

func (a Client) Release() {
	capnp.Client(a).Release()
}

func (a Client) Ls(ctx context.Context) (Iterator, capnp.ReleaseFunc) {
	f, release := api.Anchor(a).Ls(ctx, nil)
	h := &handler{Future: casm.Future(f)}
	return Iterator{
		Seq:    h,
		Future: h,
	}, release
}

// Walk to the register located at path.
func (a Client) Walk(ctx context.Context, path string) (Anchor, capnp.ReleaseFunc) {
	return walkPath(ctx, api.Anchor(a), path)
}

type Iterator casm.Iterator[Client]

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
		return nil, false
	}

	children, err := res.Children()
	if err != nil {
		h.err = err
		return nil, false
	}

	a, err := children.At(i)
	if err != nil {
		h.err = err
		return nil, false
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

// type Iterator struct {
// 	Err  error
// 	Name string
// 	pos  int
// 	cs   api.Anchor_Child_List
// }

// func newIterator(cs api.Anchor_Child_List) *Iterator {
// 	return &Iterator{cs: cs}
// }

// func newErrIterator(err error) *Iterator {
// 	return &Iterator{Err: err}
// }

// func (rs *Iterator) More() bool {
// 	return rs.Err == nil && rs.pos < rs.cs.Len()
// }

// func (rs *Iterator) Next() (more bool) {
// 	if more = rs.More(); more {
// 		rs.Name, rs.Err = rs.cs.At(rs.pos).Name()
// 		rs.pos++
// 	}

// 	return
// }

// func (rs *Iterator) Anchor() Anchor {
// 	return Anchor(rs.cs.At(rs.pos).Anchor())
// }

/*

	Generic methods for client implementations

*/

func walkPath(ctx context.Context, a api.Anchor, path Path) (Client, capnp.ReleaseFunc) {
	if path.IsRoot() {
		return Client(a), func() {}
	}

	f, release := a.Walk(ctx, walkParam(path))
	return Client(f.Anchor()), release
}

func walkParam(path Path) func(api.Anchor_walk_Params) error {
	return func(ps api.Anchor_walk_Params) error {
		return path.Bind(ps)
	}
}
