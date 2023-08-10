package query_test

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wetware/pkg/cluster/query"
	"github.com/wetware/pkg/cluster/routing"
	test_routing "github.com/wetware/pkg/cluster/routing/test"
)

func TestSelector(t *testing.T) {
	t.Parallel()
	t.Helper()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	snap := test_routing.NewMockSnapshot(ctrl)

	/*
		Test that each selector calls the expected Query method.
	*/
	for _, tt := range []struct {
		name string
		sel  query.Selector
		call *gomock.Call
	}{
		{
			name: "Match",
			sel:  query.Select(nil),
			call: snap.EXPECT().
				Get(gomock.Any()).
				Return(nil, nil).
				Times(1),
		},
		{
			name: "From",
			sel:  query.From(nil),
			call: snap.EXPECT().
				LowerBound(gomock.Any()).
				Return(nil, nil).
				Times(1),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.sel(snap)
			assert.NoError(t, err)
		})
	}
}

func TestRange(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	/*
		A range query is a constraint placed upon the `From` selector.
		It is a common (and indeed, important) pattern, and so is given
		a test of its own.
	*/

	// We define a set of records with lexicographically ordered IDs.
	// This will allow us to define a range over the id index.
	recs := []*record{
		{id: peer.ID("foo")},
		{id: peer.ID("foobar")},
		{id: peer.ID("foobarbaz")},    // <- range must support dupes!
		{id: peer.ID("foobarbaz")},    // <- last item in the range
		{id: peer.ID("foobarbazqux")}, // <- last item to be iterated upon
		{id: peer.ID("foobarbazqux")}, // <- skipped
	}

	// Create a mock iterator that is expected to iterate through all
	// but the last record.  The second-to-last record will be returned
	// by the last call to Next(), which should cause the range iterator
	// to detect that the range has been exceeded, and return nil.  Once
	// the range iterator has detected that it is out-of-bounds, it will
	// cease to call the mock iterator's Next() method. Or so we hope...!
	iter := test_routing.NewMockIterator(ctrl)
	iter.EXPECT().Next().Return(recs[0]).Times(1)
	iter.EXPECT().Next().Return(recs[1]).Times(1)
	iter.EXPECT().Next().Return(recs[2]).Times(1)
	iter.EXPECT().Next().Return(recs[3]).Times(1)
	iter.EXPECT().Next().Return(recs[4]).Times(1) // <- not returned!
	// NOTE:  the mock iterator is NOT expected to have a call to Next
	//        that returns nil! This is because the predicateIter will
	//        short-circuit it.

	// Next, we construct a query that returns the above iterator.
	snap := test_routing.NewMockSnapshot(ctrl)
	snap.EXPECT().
		LowerBound(gomock.Any()).
		Return(iter, nil).
		Times(1)

	// We now define an index over id_prefix matching 'foo'.  On its own,
	// this would match *all* records in the iterator.
	min := test_routing.NewMockIndex(ctrl)
	min.EXPECT().
		String().
		Return("id").
		AnyTimes()
	min.EXPECT().
		Prefix().
		Return(true).
		AnyTimes()

	// Now we define an index that designates the upper bound on the range.
	// This matches the highest-order id that is part of the range.
	max := test_routing.NewMockIndex(ctrl)
	max.EXPECT().
		String().
		Return("id").
		AnyTimes()
	max.EXPECT().
		Prefix().
		Return(false).
		AnyTimes()

	selector := query.Range(peerIndex(min, "foo"), peerIndex(max, "foobarbaz"))
	it, err := selector(snap)
	require.NoError(t, err, "selector should not return error")
	require.NotNil(t, it, "selector should return iterator")

	for _, want := range recs[:4] {
		r := it.Next()
		require.NotNil(t, r, "iterator should not be exhausted")
		require.Equal(t, want, r, "should be record %s", string(want.id))
	}

	require.Nil(t, it.Next(), "iterator should be exhausted")
}

type mockPeerIndex struct {
	*test_routing.MockIndex
	id string
}

func peerIndex(ix *test_routing.MockIndex, id string) mockPeerIndex {
	return mockPeerIndex{
		MockIndex: ix,
		id:        id,
	}
}

func (ix mockPeerIndex) Peer() (string, error) {
	return ix.id, nil
}

func (ix mockPeerIndex) PeerPrefix() (string, error) {
	return ix.Peer()
}

type record struct {
	once sync.Once
	id   peer.ID
	seq  uint64
	ins  uint64
	host string
	meta routing.Meta
	ttl  time.Duration
}

func (r *record) init() {
	r.once.Do(func() {
		if r.id == "" {
			r.id = newPeerID()
		}

		if r.host == "" {
			r.host = newPeerID().String()[:16]
		}

		if r.ins == 0 {
			r.ins = rand.Uint64()
		}
	})
}

func (r *record) Peer() peer.ID {
	r.init()
	return r.id
}

func (r *record) Server() routing.ID {
	r.init()
	return routing.ID(r.ins)
}

func (r *record) Seq() uint64 { return r.seq }

func (r *record) Host() (string, error) {
	r.init()
	return r.host, nil
}

func (r *record) TTL() time.Duration {
	if r.init(); r.ttl == 0 {
		return time.Second
	}

	return r.ttl
}

func (r *record) Meta() (routing.Meta, error) { return r.meta, nil }

func newPeerID() peer.ID {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	sk, _, err := crypto.GenerateEd25519Key(rnd)
	if err != nil {
		panic(err)
	}

	id, err := peer.IDFromPrivateKey(sk)
	if err != nil {
		panic(err)
	}

	return id
}
