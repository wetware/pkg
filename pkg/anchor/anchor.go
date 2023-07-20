package anchor

import (
	"context"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/ww/api/anchor"
	"github.com/wetware/ww/pkg/internal/bounded"
)

type Anchor api.Anchor

func (a Anchor) AddRef() Anchor {
	return Anchor(api.Anchor(a).AddRef())
}

func (a Anchor) Release() {
	capnp.Client(a).Release()
}

func (a Anchor) Ls(ctx context.Context) (*Iterator, capnp.ReleaseFunc) {
	f, release := api.Anchor(a).Ls(ctx, nil)
	return &Iterator{fut: f}, release
}

// Walk to the register located at path.
func (a Anchor) Walk(ctx context.Context, path string) (Anchor, capnp.ReleaseFunc) {
	p := NewPath(path)

	if p.IsRoot() {
		anchor := a.AddRef()
		return anchor, anchor.Release
	}

	f, release := api.Anchor(a).Walk(ctx, destination(p))
	return Anchor(f.Anchor()), release
}

func destination(path Path) func(api.Anchor_walk_Params) error {
	return func(ps api.Anchor_walk_Params) error {
		return path.bind(func(s string) bounded.Type[string] {
			err := ps.SetPath(trimmed(s))
			return bounded.Failure[string](err) // can be nil
		}).Err()
	}
}

type Iterator struct {
	fut api.Anchor_ls_Results_Future
	err error

	// cache
	children api.Anchor_Child_List
	index    int
}

func (it *Iterator) resolve() {
	if it.err == nil && it.children == (api.Anchor_Child_List{}) {
		var res api.Anchor_ls_Results
		if res, it.err = it.fut.Struct(); it.err == nil {
			it.children, it.err = res.Children()
		}
	}
}

func (it *Iterator) Err() error {
	select {
	case <-it.fut.Done():
		it.resolve()

	default:
	}

	return it.err
}

// Next returns the name of the next subanchor in the stream. It
// returns an empty string when the iterator has been exhausted.
func (it *Iterator) Next() (name string) {
	if it.children == (api.Anchor_Child_List{}) {
		it.resolve()
	} else {
		it.index++
	}

	if it.more() {
		name, it.err = it.children.At(it.index).Name()
	}

	return
}

func (it *Iterator) more() bool {
	size := it.children.Len()
	return it.err == nil && it.index < size
}

func (it *Iterator) Anchor() Anchor {
	if it.more() {
		return Anchor(it.children.At(it.index).Anchor())
	}

	return Anchor{}
}
