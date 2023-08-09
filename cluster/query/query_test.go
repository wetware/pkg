package query_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wetware/ww/cluster/query"
	"github.com/wetware/ww/cluster/routing"
	mock_routing "github.com/wetware/ww/internal/mock/cluster/routing"
)

func TestQuery(t *testing.T) {
	t.Parallel()
	t.Helper()

	recs := []*record{
		{id: newPeerID()},
		{id: newPeerID()},
		{id: newPeerID()},
		{id: newPeerID()},
		{id: newPeerID()},
	}

	t.Run("Lookup", func(t *testing.T) {
		t.Parallel()
		t.Helper()

		t.Run("Forward", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			/*
				For the forward lookup test, we'll actually check that
				the method call returns the expected result.   For the
				reverse lookup test, we'll take a simplified approach.
			*/

			iter := mock_routing.NewMockIterator(ctrl)
			iter.EXPECT().
				Next().
				Return(recs[0]).
				Times(1)

			snap := mock_routing.NewMockSnapshot(ctrl)
			snap.EXPECT().
				Get(gomock.Any()).
				Return(iter, nil).
				Times(1)

			// Double-reverse so that we test the code path that
			// takes us *back* to a normal, forward-iterating query.
			v := query.Query{Snapshot: snap}.Reverse().Reverse()

			r, err := v.Lookup(query.All())
			require.NoError(t, err, "lookup should succeed")
			require.NotNil(t, r, "should return record")

			want := routing.Record(recs[0])
			assert.Equal(t, want, r, "should match %s", recs[0].Peer())
		})

		t.Run("Reverse", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			/*
				Just test that reversing the query calls the expected query method.
				In addition to simplifying test code, it explores the code path in
				which we abort if a nil iterator is returned the internal call to
				v.Iter()
			*/

			snap := mock_routing.NewMockSnapshot(ctrl)
			snap.EXPECT().
				GetReverse(gomock.Any()). // <- what we're actually testing
				Return(nil, nil).
				Times(1)

			v := query.Query{Snapshot: snap}.Reverse()

			_, err := v.Lookup(query.All())
			require.NoError(t, err, "lookup should succeed")
		})
	})

	t.Run("Iter", func(t *testing.T) {
		t.Parallel()
		t.Helper()

		t.Run("Forward", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			/*
				For the forward iteration test, we'll actually check that
				the iterator covers the expect range of records.  For the
				reverse iteration test, we'll take a simplified approach.
			*/

			// Iterate through records once, then return nil
			iter := mock_routing.NewMockIterator(ctrl)
			for _, r := range recs {
				iter.EXPECT().Next().Return(r).Times(1)
			}
			iter.EXPECT().Next().Return(nil).Times(1)

			snap := mock_routing.NewMockSnapshot(ctrl)
			snap.EXPECT().
				Get(gomock.Any()).
				Return(iter, nil).
				Times(1)

			// Double-reverse so that we test the code path that
			// takes us *back* to a normal, forward-iterating query.
			v := query.Query{Snapshot: snap}.Reverse().Reverse()

			it, err := v.Iter(query.All())
			require.NoError(t, err, "lookup should succeed")
			require.NotNil(t, it, "should return iterator")

			var ctr int
			for r := it.Next(); r != nil; r = it.Next() {
				want := routing.Record(recs[ctr])
				require.Equal(t, want, r, "should match record %d", ctr)
				ctr++
			}
		})

		t.Run("Reverse", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			/*
				Just test that reversing the query calls the expected query method.
				In addition to simplifying test code, it explores the code path in
				which a nil iterator is produced by the Selector.
			*/

			snap := mock_routing.NewMockSnapshot(ctrl)
			snap.EXPECT().
				GetReverse(gomock.Any()).
				Return(nil, nil).
				Times(1)

			v := query.Query{Snapshot: snap}.Reverse()

			_, err := v.Iter(query.All())
			require.NoError(t, err, "lookup should succeed")
		})
	})
}
