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
	"github.com/wetware/ww/pkg/vat/cap/cluster"
)

func TestRecord(t *testing.T) {
	t.Parallel()

	assert.Error(t, cluster.Record{}.Validate(),
		"should not pass validation with empty ID field")

	assert.Panics(t, func() { _ = cluster.Record{}.Peer() },
		"should panic on empty Peer field")

	id, err := cluster.Record{}.ID()
	assert.Zero(t, id,
		"zero-value Record should produce zero-value peer.ID")
	assert.Error(t, err,
		"should report validation error")
}

func TestMultipleClients(t *testing.T) {
	t.Parallel()

	const N = 10

	server := &cluster.ViewServer{RoutingTable: make(routingTable, 65)}
	for i := 0; i < N; i++ {
		require.NotEmpty(t, server.Client())
	}
}

func TestIter(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var (
			rt = routingTable(nil)
			c  = cluster.View{
				Client: cluster.ViewServer{RoutingTable: rt}.Client(),
			}
		)

		it := c.Iter(ctx)

		require.Nil(t, it.Record(), "should be exhausted")
		require.NoError(t, it.Err, "should not fail")
	})

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

		c := cluster.View{
			Client: cluster.ViewServer{RoutingTable: rt}.Client(),
		}

		it := c.Iter(ctx)
		it.Next()

		require.NotNil(t, it.Record(), "should not be exhausted")
		require.NoError(t, it.Err, "should succeed")

		it.Next()

		require.Nil(t, it.Record(), "should be exhausted")
		assert.NoError(t, it.Err, "should not fail")
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

		var (
			i  int
			it *cluster.RecordStream
			c  = cluster.View{
				Client: cluster.ViewServer{RoutingTable: rt}.Client(),
			}
		)

		for it = c.Iter(ctx); it.Record() != nil; it.Next() {
			require.NoError(t, it.Err)

			require.NotNil(t, it.Record(),
				"should not be exhausted")
			require.False(t, it.Deadline().IsZero(),
				"should have nonzero deadline")

			require.Equal(t, rt[i].Peer(), it.Record().Peer(),
				"should match peer.ID at index %d", i)
			require.Equal(t, uint64(i), it.Record().Seq(),
				"should have match sequence at index %d", i)

			i++
		}

		assert.NoError(t, it.Err, "should not fail")
	})

	t.Run("Finish", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var (
			rt = blockingRoutingTable{ctx}
			c  = cluster.View{
				Client: cluster.ViewServer{RoutingTable: rt}.Client(),
			}
		)

		it := c.Iter(ctx)
		it.Finish()

		require.Nil(t, it.Record(),
			"should be exhausted after context cancellation")
	})
}

func TestLookup(t *testing.T) {
	t.Parallel()
	t.Helper()

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

	c := cluster.View{
		Client: cluster.ViewServer{RoutingTable: rt}.Client(),
	}

	t.Run("Exists", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		want := rt[42]

		f, release := c.Lookup(ctx, want.id)
		require.NotZero(t, f, "should return FutureRecord")
		require.NotNil(t, release, "should return ReleaseFunc")
		defer release()

		got, err := f.Await(ctx)
		require.NoError(t, err, "should resolve successfully")
		require.NotZero(t, got, "should return Record")

		assert.Equal(t, want.Peer(), got.Peer())
		assert.Equal(t, got.Seq(), want.seq)
		assert.Greater(t, got.TTL(), time.Duration(0))
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		f, release := c.Lookup(ctx, newID())
		require.NotZero(t, f, "should return FutureRecord")
		require.NotNil(t, release, "should return ReleaseFunc")
		defer release()

		got, err := f.Await(ctx)
		assert.Zero(t, got, "should return zero-value Record")
		assert.ErrorIs(t, err, cluster.ErrNotFound, "should return ErrNotFound")
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
