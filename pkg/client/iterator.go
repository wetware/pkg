package client

import (
	"context"

	"github.com/pkg/errors"
	capnp "zombiezen.com/go/capnproto2"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/lthibault/wetware/internal/api"
	ww "github.com/lthibault/wetware/pkg"
)

var (
	_ ww.Iterator = (*clusterIterator)(nil)
	_ callHandler = (*clusterIterator)(nil)

	_ ww.Iterator = (*anchorIterator)(nil)
	_ ww.Iterator = (*refIterator)(nil)
	_ ww.Iterator = (*emptyIterator)(nil)
	_ ww.Iterator = (*errorIterator)(nil)
)

// clusterIterator is an RPC handler for Anchor.Ls
// It implements ww.Iterator and is suitable for use as Anchor.Ls's return value.
type clusterIterator struct {
	ps  peer.IDSlice
	idx int

	err  error
	term terminal
}

func (it *clusterIterator) Reset() {
	it.idx = -1
	it.err = nil
	it.ps = nil
}

func (it *clusterIterator) Fail(err error) {
	it.err = err
}

func (it *clusterIterator) HandleRPC(ctx context.Context, s network.Stream) error {
	defer s.Close()
	it.Reset()

	t, _ := ctx.Deadline()
	if err := s.SetReadDeadline(t); err != nil {
		return errors.Wrap(err, "set deadline")
	}

	msg, err := capnp.NewPackedDecoder(s).Decode()
	if err != nil {
		return errors.Wrap(err, "decode hosts")
	}

	ps, err := api.ReadRootPeerSet(msg)
	if err != nil {
		return errors.Wrap(err, "read root peerset")
	}

	ids, err := ps.Ids()
	if err != nil {
		return errors.Wrap(err, "read host IDs")
	}

	it.ps = make(peer.IDSlice, ids.Len())
	for i := range it.ps {
		raw, err := ids.At(i)
		if err != nil {
			return errors.Wrapf(err, "error reading id at index %d", i)
		}

		if it.ps[i], err = peer.Decode(raw); err != nil {
			return errors.Wrapf(err, "malformed id at index %d", i)
		}
	}

	return nil
}

func (it clusterIterator) Err() error {
	return it.err
}

func (it *clusterIterator) Next() (more bool) {
	if more = it.more(); more {
		it.idx++
	}

	return
}

func (it clusterIterator) Path() string {
	return it.id().String()
}

func (it clusterIterator) Anchor() ww.Anchor {
	return &lazyAnchor{
		sess: it.term.Dial(it.id()),
	}
}

func (it clusterIterator) more() bool {
	return it.err == nil && it.ps != nil && it.idx < len(it.ps)-1
}

func (it clusterIterator) id() peer.ID {
	return it.ps[it.idx]
}

type anchorIterator struct {
	cs api.Anchor_SubAnchor_List

	idx int
	err error
}

func newAnchorIterator(cs api.Anchor_SubAnchor_List) ww.Iterator {
	if !cs.HasData() || cs.Len() == 0 {
		return emptyIterator{}
	}

	return &anchorIterator{
		cs:  cs,
		idx: -1,
	}
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

type refIterator struct {
	ww.Iterator
	ref interface{}
}

func (it refIterator) Anchor() ww.Anchor {
	return refAnchor{
		Anchor: it.Iterator.Anchor(),
		ref:    it.ref,
	}
}

type emptyIterator struct{}

func (emptyIterator) Err() error        { return nil }
func (emptyIterator) Next() bool        { return false }
func (emptyIterator) Path() string      { return "" }
func (emptyIterator) Anchor() ww.Anchor { return nil }

type errorIterator struct {
	error
	emptyIterator
}

func errIter(err error) errorIterator {
	return errorIterator{error: err}
}

func (it errorIterator) Err() error { return it.error }
