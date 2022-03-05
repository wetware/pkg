package cluster_test

import (
	"context"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wetware/casm/pkg/cluster/routing"
	"github.com/wetware/ww/pkg/cap/cluster"
)

func TestIter(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("Single", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		rt := routingTable{
			{
				id:  newID(),
				ttl: time.Second * 10,
				seq: 42,
				dl:  time.Now().Add(time.Second * 10),
			},
		}

		c := (&cluster.ViewServer{View: rt}).NewClient(nil)

		it, release := c.Iter(ctx)
		defer release()

		ok := it.Next(ctx)
		require.True(t, ok, "should advance iterator")
		require.NoError(t, it.Err, "should succeed")

		assert.NotZero(t, it.Record())

		ok = it.Next(ctx)
		require.False(t, ok, "should not advance iterator")
		assert.NoError(t, it.Err, "should be exhausted")
	})

	t.Run("Batch", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		dl := time.Now().Add(time.Second * 10)
		var rt = make(routingTable, 65)
		for i := range rt {
			rt[i] = record{
				id:  newID(),
				ttl: time.Second * 10,
				seq: uint64(i),
				dl:  dl,
			}
		}

		c := (&cluster.ViewServer{View: rt}).NewClient(nil)

		it, release := c.Iter(ctx)
		defer release()

		for i := 0; it.Next(ctx); i++ {
			require.NoError(t, it.Err)

			r := it.Record()
			require.NotNil(t, r)
			require.Equal(t, rt[i].Peer(), r.Peer())
			require.Equal(t, uint64(i), r.Seq())
			require.Greater(t, r.TTL(), time.Duration(0),
				"should have positive, nonzero TTL")
		}

		assert.NoError(t, it.Err, "should be exhausted")
	})

	t.Run("Cancel", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var (
			rt = blockingRoutingTable{ctx}
			c  = (&cluster.ViewServer{View: rt}).NewClient(nil)
		)

		it, release := c.Iter(ctx)
		defer release()

		ctx, cancel = context.WithCancel(ctx)
		cancel()

		assert.Eventually(t, func() bool {
			return !it.Next(ctx) && errors.Is(it.Err, context.Canceled)
		}, time.Second, time.Millisecond*10,
			"should eventually report context.Canceled")
	})
}

func TestLookup(t *testing.T) {
	t.Parallel()
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dl := time.Now().Add(time.Second * 10)
	var view = make(routingTable, 65)
	for i := range view {
		view[i] = record{
			id:  newID(),
			ttl: time.Second * 10,
			seq: uint64(i),
			dl:  dl,
		}
	}

	c := (&cluster.ViewServer{View: view}).NewClient(nil)

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

type blockingRoutingTable struct{ ctx context.Context }

func (b blockingRoutingTable) Iter() routing.Iterator {
	return blockingRoutingTable{b.ctx}
}

func (b blockingRoutingTable) Lookup(id peer.ID) (routing.Record, bool) {
	<-b.ctx.Done()
	return nil, false
}

func (b blockingRoutingTable) Next()                { <-b.ctx.Done() }
func (blockingRoutingTable) Record() routing.Record { return nil }
func (blockingRoutingTable) Deadline() time.Time    { return time.Time{} }
func (blockingRoutingTable) Finish()                {}

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

type routingTable []record

func (v routingTable) Iter() routing.Iterator {
	return &iter{
		recs: v,
	}
}

func (v routingTable) Lookup(id peer.ID) (routing.Record, bool) {
	for _, r := range v {
		if r.Peer() == id {
			return r, true
		}
	}

	return nil, false
}
