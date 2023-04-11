package anchor

import (
	"context"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/ww/internal/api/anchor"
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
	return &Iterator{fut: f, index: -1}, release
}

// Walk to the register located at path.
func (a Anchor) Walk(ctx context.Context, path string) (Anchor, capnp.ReleaseFunc) {
	p := NewPath(path)

	if p.IsRoot() {
		return Anchor(a), a.AddRef().Release
	}

	f, release := api.Anchor(a).Walk(ctx, destination(p))
	return Anchor(f.Anchor()), release
}

type Iterator struct {
	fut api.Anchor_ls_Results_Future
	err error

	// cache
	children api.Anchor_Child_List
	index    int
}

func (it *Iterator) Err() error {
	if it.err == nil {
		select {
		case <-it.fut.Done():
			var res api.Anchor_ls_Results
			if res, it.err = it.fut.Struct(); it.err != nil {
				break
			}

			if it.children, it.err = res.Children(); it.err != nil {
				break
			}

		default:
		}
	}

	return it.err
}

// Next returns the name of the next subanchor in the stream. It
// returns an empty string when the iterator has been exhausted.
func (it *Iterator) Next() (name string) {
	if it.Err() == nil {
		it.index++
		name, it.err = it.children.At(it.index).Name()
	}

	return
}

func (it Iterator) Anchor() Anchor {
	// it.Err() was called by Next(), so there's no point in
	// doing all the checks again.
	if it.err == nil {
		return Anchor(it.children.At(it.index).Anchor())
	}

	return Anchor{}
}

func destination(path Path) func(api.Anchor_walk_Params) error {
	return func(ps api.Anchor_walk_Params) error {
		return path.bind(func(s string) bounded.Type[string] {
			err := ps.SetPath(trimmed(s))
			return bounded.Failure[string](err) // can be nil
		}).Err()
	}
}

/*---------------------------*
|                            |
|    Server Implementation   |
|                            |
*----------------------------*/

type server struct{ *node }

func (s server) Shutdown() {
	// anchorRef holds the lock when shutting down.
	s.Release()
}

func (s server) Ls(ctx context.Context, call api.Anchor_ls) error {
	s.Lock()
	defer s.Unlock()

	children := s.children

	if len(children) == 0 {
		return nil
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	cs, err := res.NewChildren(int32(len(children)))
	if err != nil {
		return err
	}

	var index int
	for name, child := range children {
		if err = cs.At(index).SetName(name); err != nil {
			break
		}

		if err = cs.At(index).SetAnchor(child.Anchor()); err != nil {
			break
		}

		index++
	}

	return err
}

// FIXME:  there is currently a vector for resource-exhaustion attacks.
// We don't enforce a maximum depth on anchors, nor do we enforce a max
// number of children per node. An attacker can exploit this by walking
// an arbitrarily long path and/or by creating arbitrarily many anchors,
// ultimately exhausting the attacker's memory.
func (s server) Walk(ctx context.Context, call api.Anchor_walk) error {
	path := newPath(call)
	if path.Err() != nil {
		return path.Err()
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	// If path is root, just increment the refcount for n and return a
	// new anchor client.
	if path.IsRoot() {
		return res.SetAnchor(s.AddRef().Anchor())
	}

	// Iteratively "walk" to designated path.  It's important to avoid
	// recursion, so that RPCs can't blow up the stack.
	//
	// Each iteration of the loop shadows the n symbol, including its
	// embedded node, such that we are holding the final node when we
	// exit the loop.
	for path, name := path.Next(); name != ""; path, name = path.Next() {
		s.node = s.Child(name) // shallow copy; TODO(soon):  check for this in a unit test
	}

	return res.SetAnchor(s.Anchor())
}

func newPath(call api.Anchor_walk) Path {
	path, err := call.Args().Path()
	if err != nil {
		return failure(err)
	}

	return NewPath(path)
}
