package client

import (
	capnp "zombiezen.com/go/capnproto2"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/lthibault/wetware/internal/api"
	ww "github.com/lthibault/wetware/pkg"
)

type clusterView struct {
	view capnp.TextList
	idx  int

	id  peer.ID
	err error

	term *terminal
}

func newClusterView(term *terminal, res api.Router_ls_Results) (*clusterView, error) {
	view, err := res.View()
	if err != nil {
		return nil, err
	}

	return &clusterView{
		view: view,
		idx:  -1,
		term: term,
	}, nil
}

func (c clusterView) Err() error {
	return c.err
}

func (c *clusterView) Next() bool {
	if !c.more() {
		return false
	}

	c.idx++

	var s string
	if s, c.err = c.view.At(c.idx); c.err != nil {
		return false
	}

	if c.id, c.err = peer.Decode(s); c.err != nil {
		return false
	}

	return true
}

func (c clusterView) Path() string {
	return c.id.String()
}

func (c clusterView) Anchor() ww.Anchor {
	return &lazyAnchor{
		id:   c.id,
		term: c.term,
	}
}

func (c clusterView) more() bool {
	return c.err == nil && c.idx < c.view.Len()-1
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
