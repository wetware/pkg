package cluster_test

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wetware/casm/pkg/cluster/routing"
	"github.com/wetware/ww/pkg/cap/anchor/cluster"
)

func TestIter(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("Single", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		view := mockView{
			{
				id:  newID(),
				ttl: time.Second * 10,
				seq: 42,
				dl:  time.Now().Add(time.Second * 10),
			},
		}

		c := (&cluster.ClusterServer{view}).NewClient(nil)

		it, release := c.Iter(ctx)
		defer release()

		err := it.Next(ctx)
		assert.NoError(t, err)

		assert.NotNil(t, it.Record())
		assert.NotZero(t, it.Deadline())

		assert.ErrorIs(t, it.Next(ctx), cluster.ErrExhausted)
		assert.ErrorIs(t, it.Err, cluster.ErrExhausted)
	})

	t.Run("Batch", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		dl := time.Now().Add(time.Second * 10)
		var view = make(mockView, 65)
		for i := range view {
			view[i] = record{
				id:  newID(),
				ttl: time.Second * 10,
				seq: uint64(i),
				dl:  dl,
			}
		}

		c := (&cluster.ClusterServer{view}).NewClient(nil)

		it, release := c.Iter(ctx)
		defer release()

		for i := 0; it.Next(ctx) == nil; i++ {
			require.NoError(t, it.Err)

			r := it.Record()
			require.NotNil(t, r)
			require.Equal(t, view[i].Peer(), r.Peer())
			require.Equal(t, uint64(i), r.Seq())
			require.Greater(t, r.TTL(), time.Duration(0),
				"should have positive, nonzero TTL")
		}

		assert.ErrorIs(t, cluster.ErrExhausted, it.Err)
	})
}

func TestLookup(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("Batch", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		dl := time.Now().Add(time.Second * 10)
		var view = make(mockView, 65)
		for i := range view {
			view[i] = record{
				id:  newID(),
				ttl: time.Second * 10,
				seq: uint64(i),
				dl:  dl,
			}
		}

		c := (&cluster.ClusterServer{view}).NewClient(nil)

		want := view[42]

		f, release := c.Lookup(ctx, want.id)
		defer release()

		got, err := f.Struct()
		require.NoError(t, err)

		id, err := got.Peer()
		require.NoError(t, err)
		assert.Equal(t, want.Peer(), id)

		assert.Equal(t, got.Seq(), want.seq)
		assert.Greater(t, got.TTL(), time.Duration(0))
	})
}

func newID() peer.ID {
	pk, _, err := crypto.GenerateECDSAKeyPair(rand.Reader)
	if err != nil {
		panic(err)
	}

	id, err := peer.IDFromPrivateKey(pk)
	if err != nil {
		panic(err)
	}

	return id
}

type iter struct {
	recs []record
	idx  int
}

func (it *iter) Next() {
	it.idx++
}

func (it iter) Record() routing.Record {
	if it.idx >= len(it.recs) {
		return nil
	}

	return it.recs[it.idx]
}

func (it iter) Deadline() time.Time {
	if it.idx >= len(it.recs) {
		return time.Time{}
	}

	return it.recs[it.idx].dl
}

func (it iter) Finish() {}

type record struct {
	id  peer.ID
	ttl time.Duration
	seq uint64
	dl  time.Time
}

func (r record) Peer() peer.ID      { return peer.ID(r.id) }
func (r record) TTL() time.Duration { return r.ttl }
func (r record) Seq() uint64        { return r.seq }

type mockView []record

func (v mockView) Iter() routing.Iterator {
	return &iter{
		recs: v,
	}
}

func (v mockView) Lookup(id peer.ID) (routing.Record, bool) {
	for _, r := range v {
		if r.Peer() == id {
			return r, true
		}
	}

	return nil, false
}
