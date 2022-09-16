package anchor

import (
	"context"
	"fmt"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/ww/internal/api/anchor"
)

// Server is a shared-memory capablity.  Anchors form
// a tree where each node may have zero-or-one value, and
// zero-or-more children.  See Path for additional details
// about the tree semantics of Server.
type Server struct {
	sched Scheduler
}

// Root returns a root server anchor
func Root() Server {
	return NewAnchor(NewScheduler(root))
}

func NewAnchor(sched Scheduler) Server {
	return Server{
		sched: sched,
	}
}

func (s Server) Anchor() Anchor {
	return Anchor(api.Anchor_ServerToClient(s))
}

func (s Server) Client() capnp.Client {
	return capnp.Client(s.Anchor())
}

func (s Server) Shutdown() {
	// Optimistic strategy:  first check if a should be scrubbed using a
	//                       read-only transaction. Acquire the lock iff
	//                       a scrub takes place.
	if rx := s.sched.Txn(false); rx.IsOrphan() {
		wx := s.sched.Txn(true)
		defer wx.Finish()

		// may have changed since we last checked
		if wx.IsOrphan() {
			_ = wx.Scrub()
			wx.Commit()
		}
	}
}

func (s Server) Name() string {
	name := s.Path().bind(last)
	return trimmed(name.String())
}

func (s Server) Path() Path {
	return s.sched.root
}
func (s Server) Ls(ctx context.Context, call api.Anchor_ls) error {
	tx := s.sched.Txn(false)

	it, err := tx.Children()
	if err != nil {
		return err
	}

	var children []Server
	for v := it.Next(); v != nil; v = it.Next() {
		children = append(children, v.(Server))
	}

	// skip allocation if there are no children
	if len(children) == 0 {
		return nil
	}

	return bindLs(call, children)
}

func (s Server) Walk(ctx context.Context, call api.Anchor_walk) error {
	raw, err := call.Args().Path()
	if err != nil {
		return err
	}

	path := NewPath(raw)
	if path.Err() != nil {
		return fmt.Errorf("path: %w", path.Err())
	}

	// Optimistic walk.  We first try with a read-only transaction, and
	// only create a write-transaction if we need to create subpaths.
	tx := s.sched.Txn(false)

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
			subanchor = s
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

	if err = bindWalk(call, subanchor); err == nil {
		tx.Commit()
	}

	return err
}

// ensurePath traverses the path and construts any missing anchors along
// the way.  The argument 'tx' MUST be a write transaction.  Callers are
// are responsible for calling Commit(), Abort() or Finish().
func (s Server) ensurePath(tx Txn, path Path) (_ Server, err error) {
	for p, name := path.Next(); name != ""; p, name = p.Next() {
		child := s.Path().WithChild(name)
		if s, err = tx.GetOrCreate(child); err != nil {
			break
		}
	}

	return s, err
}

func bindLs(call api.Anchor_ls, cs []Server) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	size := int32(len(cs))

	names, err := res.NewNames(size)
	if err != nil {
		return err
	}

	children, err := res.NewChildren(size)
	if err != nil {
		return err
	}

	for i, c := range cs {
		if err := names.Set(i, c.Name()); err != nil {
			return err
		}

		if err := children.Set(i, anchor(c)); err != nil {
			return err
		}
	}

	return err
}

func bindWalk(call api.Anchor_walk, s Server) error {
	res, err := call.AllocResults()
	if err == nil {
		err = res.SetAnchor(anchor(s))
	}

	return err
}

func anchor(c Server) api.Anchor {
	return api.Anchor(c.Client())
}
