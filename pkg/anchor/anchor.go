package anchor

import (
	"context"

	"capnproto.org/go/capnp/v3"
	casm "github.com/wetware/casm/pkg"
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

func (a Anchor) Ls(ctx context.Context) (Iterator, capnp.ReleaseFunc) {
	f, release := api.Anchor(a).Ls(ctx, nil)

	h := &sequence{
		fut: f,
		pos: -1,
	}

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

	f, release := api.Anchor(a).Walk(ctx, destination(p))
	return Anchor(f.Anchor()), release
}

type Child struct {
	index int
	res   api.Anchor_ls_Results
}

func (c Child) Name() (string, error) {
	names, err := c.res.Names()
	if err != nil {
		return "", err
	}

	return names.At(c.index)
}

func (c Child) Anchor() (Anchor, error) {
	children, err := c.res.Children()
	if err != nil {
		return Anchor{}, err
	}

	a, err := children.At(c.index)
	return Anchor(a), err
}

type Iterator casm.Iterator[Child]

func (it Iterator) Next() Child {
	c, _ := it.Seq.Next()
	return c
}

type sequence struct {
	fut api.Anchor_ls_Results_Future
	err error
	pos int
}

func (seq sequence) Done() <-chan struct{} {
	return seq.fut.Done()
}

func (seq *sequence) Err() error {
	if seq.err == nil {
		select {
		case <-seq.fut.Done():
			_, seq.err = seq.fut.Struct()
		default:
		}
	}

	return seq.err
}

func (seq *sequence) Next() (c Child, ok bool) {
	if ok = seq.Err() == nil; ok {
		seq.pos++

		c.index = seq.pos
		c.res, _ = seq.fut.Struct() // error was checked by seq.Err()
	}

	return
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

func (n node) Shutdown() {
	// anchorRef holds the lock when shutting down.
	n.Value().client.Release()
}

func (server) Ls(ctx context.Context, call api.Anchor_ls) error {
	panic("NOT IMPLEMENTED")
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

/*
	...
*/
