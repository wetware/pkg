package query_test

import (
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"

	"github.com/wetware/ww/cluster/query"
	"github.com/wetware/ww/cluster/routing"
	mock_routing "github.com/wetware/ww/internal/mock/cluster/routing"
)

func TestFirst(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	recs := []*record{
		{id: peer.ID("foo")},
		{id: peer.ID("foobar")},
	}

	iter := mock_routing.NewMockIterator(ctrl)
	iter.EXPECT().Next().Return(recs[0]).Times(1)
	iter.EXPECT().Next().Return(recs[1]).Times(1) // <- skipped

	snap := mock_routing.NewMockSnapshot(ctrl)
	snap.EXPECT().
		Get(gomock.Any()). // <- what we're actually testing
		Return(iter, nil).
		Times(1)

	selector := query.All().Bind(query.First())
	it, err := selector(snap)
	require.NoError(t, err)
	require.NotNil(t, it)

	require.Equal(t, routing.Record(recs[0]), it.Next(),
		"should return first record in iterator")
	require.Nil(t, it.Next(),
		"iterator should be expired")

}

func TestLimit(t *testing.T) {
	t.Parallel()

	/*
		Normal operation of Limit is already tested by TestFirst.
		Here, we just test the code path corresponding to invalid
		parameters.
	*/

	it, err := query.Limit(0)(nil)(nil)
	require.Error(t, err, "should reject invalid limit '0'")
	require.Nil(t, it, "should not return an iterator")

	it, err = query.Limit(-1)(nil)(nil)
	require.Error(t, err, "should reject invalid limit '0'")
	require.Nil(t, it, "should not return an iterator")
}

func TestWhere(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	recs := []*record{
		{id: peer.ID("foo")},
		{id: peer.ID("foobar")},
		{id: peer.ID("foobarbaz")},
		{id: peer.ID("quxbazbar")},
	}

	iter := mock_routing.NewMockIterator(ctrl)
	iter.EXPECT().Next().Return(recs[0]).Times(1) // <- skipped
	iter.EXPECT().Next().Return(recs[1]).Times(1)
	iter.EXPECT().Next().Return(recs[2]).Times(1) // <-skipped
	iter.EXPECT().Next().Return(recs[3]).Times(1)
	iter.EXPECT().Next().Return(nil).Times(1) // <- exhausted

	snap := mock_routing.NewMockSnapshot(ctrl)
	snap.EXPECT().
		Get(gomock.Any()). // <- what we're actually testing
		Return(iter, nil).
		Times(1)

	containsBar := matchFunc(func(r routing.Record) bool {
		return strings.Contains(string(r.Peer()), "bar")
	})

	selector := query.All().Bind(query.Where(containsBar))

	it, err := selector(snap)
	require.NoError(t, err)
	require.NotNil(t, it)

	var got []routing.Record
	for r := it.Next(); r != nil; r = it.Next() {
		got = append(got, r)
	}

	require.Len(t, got, 3)
	require.NotContains(t, got, recs[0])
}

type matchFunc func(routing.Record) bool

func (match matchFunc) Match(r routing.Record) bool {
	return match(r)
}
