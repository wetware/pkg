package client

import (
	"context"
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"
	syncutil "github.com/lthibault/util/sync"
	"github.com/lthibault/wetware/internal/api"
	ww "github.com/lthibault/wetware/pkg"
	anchorpath "github.com/lthibault/wetware/pkg/util/anchor/path"
)

type anchor struct{ api.Anchor }

func (a anchor) Ls(ctx context.Context) (ww.Iterator, error) {
	res, err := a.Anchor.Ls(ctx, func(p api.Anchor_ls_Params) error {
		return nil
	}).Struct()
	if err != nil {
		return nil, err
	}

	return newAnchorIterator(res)
}

func (a anchor) Walk(ctx context.Context, path []string) (ww.Anchor, error) {
	res, err := a.Anchor.Walk(ctx, func(param api.Anchor_walk_Params) error {
		return param.SetPath(anchorpath.Join(path))
	}).Struct()
	if err != nil {
		return nil, err
	}

	return anchor{res.Anchor()}, nil
}

type anchorIterator struct {
	cs api.Anchor_SubAnchor_List

	idx int
	err error
}

func newAnchorIterator(res api.Anchor_ls_Results) (ww.Iterator, error) {
	cs, err := res.Children()
	if err != nil {
		return nil, err
	}

	return &anchorIterator{
		cs:  cs,
		idx: -1,
	}, nil
}

func (it anchorIterator) Err() error {
	return it.err
}

func (it *anchorIterator) Next() (more bool) {
	if more = it.more(); more {
		it.idx++
	}

	return
}

func (it *anchorIterator) Path() (s string) {
	if it.err != nil {
		return
	}

	s, it.err = it.subanchor().Path()
	return
}

func (it *anchorIterator) Anchor() ww.Anchor {
	if it.err != nil {
		return nil
	}

	// TODO:  manage lifecycle
	return &anchor{it.subanchor().Anchor()}
}

func (it anchorIterator) subanchor() api.Anchor_SubAnchor {
	return it.cs.At(it.idx)
}

func (it anchorIterator) more() bool {
	return it.err == nil && it.idx < it.cs.Len()-1
}

// lazyAnchor is effectively an anchor that is not yet connected to a host.
// This is needed because ww.Iterator.Anchor() takes no argument, yet a context is
// needed in order to dial out to the remote host.  As such, we defer dialing until a
// call to one of lazyAnchor's methods is made.
type lazyAnchor struct {
	id   peer.ID
	term *terminal
	anchor

	flag syncutil.Flag
	mu   sync.Mutex
}

// ensureConnection is effectively a specialized implementation of sync.Once.Do that
// ensures exactly one connection to a remote host is dialed.  If a connection attempt
// succeeds, ensureConnection returns nil, and subsequent calls are nops.
//
// For the avoidance of doubt:  calling ensureConnection after it has returned a non-nil
// error is legal, and will attempt to connect to the remote host.
//
// ensureConnection is thread-safe.
func (la *lazyAnchor) ensureConnection(ctx context.Context) {
	if la.flag.Bool() {
		// a previous call completed successfully
		return
	}

	la.mu.Lock()
	defer la.mu.Unlock()

	// we hold the lock, so we can access fields directly.
	if la.flag != 0 {
		// a concurrent call completed successfully while
		// we were waiting for the lock
		return
	}

	la.Client = la.term.Dial(ctx, la.id)
	la.flag.Set()

	return
}

func (la *lazyAnchor) Ls(ctx context.Context) (ww.Iterator, error) {
	la.ensureConnection(ctx)
	return la.anchor.Ls(ctx)
}

func (la *lazyAnchor) Walk(ctx context.Context, path []string) (_ ww.Anchor, err error) {
	la.ensureConnection(ctx)
	return la.anchor.Walk(ctx, path)
}

type lsResults struct {
	idx    int
	err    error
	as     api.Anchor_SubAnchor_List
	id     peer.ID
	anchor api.Anchor

	term *terminal
}

func newLsResults(term *terminal, res api.Anchor_ls_Results) (*lsResults, error) {
	as, err := res.Children()
	if err != nil {
		return nil, err
	}

	return &lsResults{
		as:   as,
		idx:  -1,
		term: term,
	}, nil
}

func (c *lsResults) Err() error {
	return c.err
}

func (c *lsResults) Next() (more bool) {
	if more = c.more(); more {
		c.idx++
		c.decodeCurrentID()
		more = c.err == nil
	}

	return
}

func (c *lsResults) Path() string {
	return c.id.String()
}

func (c *lsResults) Anchor() ww.Anchor {
	if !c.isRootAnchor() {
		return anchor{c.current().Anchor()}
	}

	return &lazyAnchor{
		id:   c.id,
		term: c.term,
	}
}

func (c *lsResults) current() api.Anchor_SubAnchor {
	return c.as.At(c.idx)
}

func (c *lsResults) isRootAnchor() bool {
	return c.current().Which() == api.Anchor_SubAnchor_Which_root
}

func (c *lsResults) decodeCurrentID() {
	var s string
	if s, c.err = c.current().Path(); c.err != nil {
		return
	}

	if c.id, c.err = peer.Decode(s); c.err != nil {
		return
	}

	return
}

func (c *lsResults) more() bool {
	return c.err == nil && c.idx < c.as.Len()-1
}
