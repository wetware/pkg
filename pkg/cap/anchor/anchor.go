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

// AnchorServer is a shared-memory capablity.  Anchors form
// a tree where each node may have zero-or-one value, and
// zero-or-more children.  See Path for additional details
// about the tree semantics of AnchorServer.
type AnchorServer struct {
	sched  Scheduler
	anchor anchor.Anchor_Server
	*server.Policy
}

// Root returns a root server anchor
func Root(a anchor.Anchor_Server) AnchorServer {
	return NewAnchor(NewScheduler(root), a)
}

func NewAnchor(sched Scheduler, a anchor.Anchor_Server) AnchorServer {
	return AnchorServer{
		sched:  sched,
		anchor: a,
	}
}

func (a AnchorServer) Client() *capnp.Client {
	switch s := a.anchor.(type) {
	case *AnchorServer, AnchorServer:
		c := anchor.Anchor_ServerToClient(s, a.Policy)
		return c.Client

	case vat.ClientProvider:
		return s.Client()

	default:
		c := anchor.Anchor_ServerToClient(s, a.Policy)
		return c.Client
	}
}

func (a AnchorServer) Shutdown() {
	// Optimistic strategy:  first check if a should be scrubbed using a
	//                       read-only transaction. Acquire the lock iff
	//                       a scrub takes place.
	if rx := a.sched.Txn(false); rx.IsOrphan() {
		wx := a.sched.Txn(true)
		defer wx.Finish()

		// may have changed since we last checked
		if wx.IsOrphan() {
			_ = wx.Scrub()
			wx.Commit()
		}
	}
}

func (a AnchorServer) Name() string {
	name := a.Path().bind(last)
	return trimmed(name.String())
}

func (a AnchorServer) Path() Path {
	return a.sched.root
}

func (a AnchorServer) Ls(ctx context.Context, call anchor.Anchor_ls) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	tx := a.sched.Txn(false)

	it, err := tx.Children()
	if err != nil {
		return err
	}

	var children []AnchorServer
	for v := it.Next(); v != nil; v = it.Next() {
		children = append(children, v.(AnchorServer))
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

func (a AnchorServer) Walk(ctx context.Context, call anchor.Anchor_walk) error {
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
func (a AnchorServer) ensurePath(tx Txn, path Path) (_ AnchorServer, err error) {
	for p, name := path.Next(); name != ""; p, name = p.Next() {
		child := a.Path().WithChild(name)
		if a, err = tx.GetOrCreate(child); err != nil {
			break
		}
	}

	return a, err
}

func (a AnchorServer) Bind(target AnchorSetter) (err error) {
	anchor := anchor.Anchor{Client: a.Client()}
	if err = target.SetAnchor(anchor); err != nil {
		return
	}

	if s, ok := target.(NameSetter); ok {
		err = s.SetName(a.Name())
	}

	return
}
