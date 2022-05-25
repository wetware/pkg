package anchor

import (
	"context"
	"fmt"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	"github.com/wetware/ww/internal/api/anchor"
	"github.com/wetware/ww/pkg/vat"
)

// AnchorSetter represents any type that can that can
// receive an anchor capability.
type AnchorSetter interface {
	SetAnchor(anchor.Anchor) error
}

// NameSetter is an optional interface to be implemented
// by AnchorSetter.  SetName will be called by Anchor.Bind.
type NameSetter interface {
	SetName(string) error
}

// Anchor is a shared-memory capablity. Anchors form a tree
// where each node may have zero-or-one value, and may have
// zero-or-more children.   See Path for additional details
// about the tree semantics of Anchor.
type Anchor struct {
	sched  Scheduler
	anchor anchor.Anchor_Server
	*server.Policy
}

// Root returns a root server anchor
func Root(a anchor.Anchor_Server) Anchor {
	return NewAnchor(NewScheduler(root), a)
}

func NewAnchor(sched Scheduler, a anchor.Anchor_Server) Anchor {
	return Anchor{
		sched:  sched,
		anchor: a,
	}
}

func (a Anchor) Client() *capnp.Client {
	switch s := a.anchor.(type) {
	case *Anchor, Anchor:
		c := anchor.Anchor_ServerToClient(s, a.Policy)
		return c.Client

	case anchor.Host_Server:
		h := anchor.Host_ServerToClient(s, a.Policy)
		return h.Client

	case vat.ClientProvider:
		return s.Client()

	default:
		c := anchor.Anchor_ServerToClient(s, a.Policy)
		return c.Client
	}
}

func (a Anchor) Name() string {
	name := a.Path().bind(last)
	return trimmed(name.String())
}

func (a Anchor) Path() Path {
	return a.sched.root
}

func (a Anchor) Ls(ctx context.Context, call anchor.Anchor_ls) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	tx := a.sched.Txn(false)

	it, err := tx.Children()
	if err != nil {
		return err
	}

	var children []Anchor
	for v := it.Next(); v != nil; v = it.Next() {
		children = append(children, v.(Anchor))
	}

	// skip allocation if there are no children
	if len(children) == 0 {
		return nil
	}

	cs, err := res.NewChildren(int32(len(children)))
	if err != nil {
		return err
	}

	for i, c := range children {
		if err = c.Bind(cs.At(i)); err != nil {
			return err
		}
	}

	return nil
}

func (a Anchor) Walk(ctx context.Context, call anchor.Anchor_walk) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	path := PathFromProvider(call.Args())
	if path.Err() != nil {
		return fmt.Errorf("path: %w", path.Err())
	}

	// Optimistic walk.  We first try with a read-only transaction, and
	// only create a write-transaction if we need to create subpaths.
	tx := a.sched.Txn(false)

	subanchor, err := tx.WalkLongestSubpath(path)
	if err != nil {
		return err
	}

	// This leaves the path unchanged if subanchor.Path().IsZero().
	remain := subanchor.Path().bind(trimPrefix(path))

	// Not an exact match?
	if subanchor.Path().IsZero() || remain.IsRoot() {

		// No match?
		if subanchor.Path().IsZero() {
			subanchor = a
		}

		// Shadows the previous read transaction.  Will be committed
		// outside of this block.
		tx = subanchor.sched.Txn(true)
		defer tx.Finish()

		// Walk the remaining path and create any missing subanchors.
		if subanchor, err = subanchor.ensurePath(tx, remain); err != nil {
			return err
		}

	}

	if err = subanchor.Bind(res); err == nil {
		tx.Commit()
	}

	return err
}

// ensurePath traverses the path and construts any missing anchors along
// the way.  The argument 'tx' MUST be a write transaction.  Callers are
// are responsible for calling Commit(), Abort() or Finish().
func (a Anchor) ensurePath(tx Txn, path Path) (_ Anchor, err error) {
	for p, name := path.Next(); name != ""; p, name = p.Next() {
		child := a.Path().WithChild(name)
		if a, err = tx.GetOrCreate(child); err != nil {
			break
		}
	}

	return a, err
}

func (a Anchor) Bind(target AnchorSetter) (err error) {
	anchor := anchor.Anchor{Client: a.Client()}
	if err = target.SetAnchor(anchor); err != nil {
		return
	}

	if s, ok := target.(NameSetter); ok {
		err = s.SetName(a.Name())
	}

	return
}
