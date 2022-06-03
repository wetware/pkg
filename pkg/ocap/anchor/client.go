package anchor

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"github.com/wetware/ww/internal/api/anchor"
)

type Anchor anchor.Anchor

func (a Anchor) AddRef() Anchor {
	return Anchor(anchor.Anchor(a).AddRef())
}

func (a Anchor) Release() {
	a.Client.Release()
}

func (a Anchor) Ls(ctx context.Context) (*Iterator, capnp.ReleaseFunc) {
	return listChildren(ctx, anchor.Anchor(a))
}

// Walk to the register located at path.
func (a Anchor) Walk(ctx context.Context, path Path) (Anchor, capnp.ReleaseFunc) {
	return walkPath(ctx, anchor.Anchor(a), path)
}

type Iterator struct {
	Err  error
	Name string
	pos  int
	cs   anchor.Anchor_Child_List
}

func newIterator(cs anchor.Anchor_Child_List) *Iterator {
	return &Iterator{cs: cs}
}

func newErrIterator(err error) *Iterator {
	return &Iterator{Err: err}
}

func (rs *Iterator) More() bool {
	return rs.Err == nil && rs.pos < rs.cs.Len()
}

func (rs *Iterator) Next() (more bool) {
	if more = rs.More(); more {
		rs.Name, rs.Err = rs.cs.At(rs.pos).Name()
		rs.pos++
	}

	return
}

func (rs *Iterator) Anchor() Anchor {
	return Anchor(rs.cs.At(rs.pos).Anchor())
}

/*

	Generic methods for client implementations

*/

func listChildren(ctx context.Context, a anchor.Anchor) (*Iterator, capnp.ReleaseFunc) {
	f, release := a.Ls(ctx, nil)

	res, err := f.Struct()
	if err != nil {
		release()
		return newErrIterator(err), func() {}
	}

	cs, err := res.Children()
	if err != nil {
		release()
		return newErrIterator(err), func() {}
	}

	return newIterator(cs), release
}

func walkPath(ctx context.Context, a anchor.Anchor, path Path) (Anchor, capnp.ReleaseFunc) {
	if path.IsRoot() {
		return Anchor(a), func() {}
	}

	f, release := a.Walk(ctx, walkParam(path))
	return Anchor(f.Anchor()), release
}

func walkParam(path Path) func(anchor.Anchor_walk_Params) error {
	return func(ps anchor.Anchor_walk_Params) error {
		return path.Bind(ps)
	}
}
