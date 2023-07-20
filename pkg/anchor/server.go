package anchor

import (
	"context"
	"errors"

	api "github.com/wetware/ww/api/anchor"
)

type server struct{ *Node }

func (s server) Shutdown() {
	s.Release() // nodeHook holds the lock when shutting down.
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

		if err = cs.At(index).SetAnchor(anchor(child)); err != nil {
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

	// Iteratively "walk" to designated path.  It's important to avoid
	// recursion, so that RPCs can't blow up the stack.
	//
	// Each iteration of the loop shadows the n symbol, including its
	// embedded node, such that we are holding the final node when we
	// exit the loop.
	for path, name := path.Next(); name != ""; path, name = path.Next() {
		s.Node = s.Child(name) // shallow copy
	}

	return res.SetAnchor(anchor(s))
}

func (s server) Cell(ctx context.Context, call api.Anchor_cell) error {
	return errors.New("NOT IMPLEMENTED") // TODO(soon): implement Anchor.Cell()
}

func anchor(n interface{ Anchor() Anchor }) api.Anchor {
	return api.Anchor(n.Anchor())
}

func newPath(call api.Anchor_walk) Path {
	path, err := call.Args().Path()
	if err != nil {
		return failure(err)
	}

	return NewPath(path)
}
