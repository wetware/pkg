package cluster_test

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/wetware/casm/pkg/cluster/routing"
	"github.com/wetware/ww/pkg/cap/anchor/cluster"
	// "context"
	// "math/rand"
	// "sync"
	// "time"
	// "github.com/libp2p/go-libp2p-core/host"
	// "github.com/libp2p/go-libp2p-core/peer"
	// pubsub "github.com/libp2p/go-libp2p-pubsub"
	// "github.com/stretchr/testify/assert"
	// "github.com/stretchr/testify/require"
	// "github.com/wetware/casm/pkg/cluster"
	// mx "github.com/wetware/matrix/pkg"
)

// var (
// 	nodesAmount = 5
// )

func TestIter(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	view := mockView{
		{
			id:  "testid",
			ttl: time.Second * 10,
			seq: 42,
			dl:  time.Now(),
		},
	}

	s := cluster.ClusterServer{view}

	c := s.NewClient(nil)
	it := c.Iter()

	err := it.Next(ctx)
	assert.NoError(t, err)

	assert.NotNil(t, it.Record())
	assert.NotZero(t, it.Deadline())

	err = it.Next(ctx)
	assert.ErrorIs(t, err, cluster.ErrExhausted)
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

func (r record) Peer() peer.ID      { return r.id }
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

// type Cluster struct {
// 	hs []host.Host
// 	cs []*cluster.Node
// }

// func TestClusterIter(t *testing.T) {
// 	t.Parallel()

// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	sim := mx.New(ctx)
// 	cl := newCluster(ctx, sim)
// 	defer cl.Close()

// 	s := ClusterServer{cl.cs[0].View()}
// 	c := s.NewClient(nil)

// 	assert.Eventually(t,
// 		func() bool {
// 			return len(clusterView(ctx, &c, 2)) == nodesAmount
// 		},
// 		time.Second*5,
// 		time.Millisecond*10,
// 		"peers should receive each other's bootstrap messages")
// }

// func TestClusterIterClosedUnexpected(t *testing.T) {
// 	t.Parallel()

// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	sim := mx.New(ctx)
// 	cl := newCluster(ctx, sim)
// 	defer cl.Close()

// 	assert.Eventually(t,
// 		func() bool {
// 			ctx, cancel2 := context.WithCancel(ctx)
// 			defer cancel2()

// 			s := ClusterServer{cl.cs[0].View()}
// 			c := s.NewClient(nil)

// 			it := c.Iter(1)
// 			defer it.Finish()

// 			return it.Next(ctx) != nil
// 		},
// 		time.Second*5,
// 		time.Millisecond*10,
// 		"when the server fails, the iterator should raise an error")
// }

// func TestClusterClosed(t *testing.T) {
// 	t.Parallel()

// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	sim := mx.New(ctx)
// 	cl := newCluster(ctx, sim)
// 	defer cl.Close()

// 	s := ClusterServer{cl.cs[0].View()}
// 	c := s.NewClient(nil)

// 	it := c.Iter(1)
// 	it.Finish()

// 	require.ErrorIs(t, it.Next(ctx), ErrClosed)
// }

// func TestClusterLookup(t *testing.T) {
// 	t.Parallel()

// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	cl := newCluster(ctx, mx.New(ctx))
// 	defer cl.Close()

// 	s := ClusterServer{cl.cs[0].View()}
// 	c := s.NewClient(nil)

// 	id := cl.hs[rand.Intn(nodesAmount)].ID()
// 	assert.Eventually(t,
// 		func() bool {
// 			rec, err := c.Lookup(ctx, id)
// 			return err == nil && rec != nil && rec.Peer() == peer.ID(id.String())
// 		},
// 		time.Second*8,
// 		time.Millisecond*10,
// 		"peers should receive each other's bootstrap messages")
// }

// func TestClusterLookupNotFound(t *testing.T) {
// 	t.Parallel()

// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	sim := mx.New(ctx)
// 	cl := newCluster(ctx, sim)
// 	defer cl.Close()

// 	s := ClusterServer{cl.cs[0].View()}
// 	c := s.NewClient(nil)

// 	h, err := sim.NewHost(ctx)
// 	require.NoError(t, err)

// 	id := h.ID()

// 	assert.Eventually(t,
// 		func() bool {
// 			peer, err := c.Lookup(ctx, id)
// 			return err == nil && peer == nil
// 		},
// 		time.Second*8,
// 		time.Millisecond*10,
// 		"peers should receive each other's bootstrap messages")
// }

// func newCluster(ctx context.Context, sim mx.Simulation) *Cluster {
// 	var (
// 		cl = Cluster{
// 			hs: make([]host.Host, nodesAmount),
// 			cs: make([]*cluster.Node, nodesAmount),
// 		}
// 		wg sync.WaitGroup
// 	)

// 	// init hosts
// 	for i := 0; i < nodesAmount; i++ {
// 		wg.Add(1)
// 		go func(id int) {
// 			defer wg.Done()

// 			cl.hs[id] = sim.MustHost(ctx)
// 		}(i)
// 	}
// 	wg.Wait()

// 	// init cluster nodes
// 	for i := 0; i < nodesAmount; i++ {
// 		wg.Add(1)
// 		go func(id int) {
// 			defer wg.Done()

// 			// decide neighbors to bootstrap pubsub
// 			id1, id2 := id-1, id+1
// 			if id1 < 0 {
// 				id1 = nodesAmount - 1
// 			}
// 			if id2 >= nodesAmount {
// 				id2 = 0
// 			}

// 			// init pubsub + cluster node
// 			ps, err := pubsub.NewGossipSub(ctx, cl.hs[id],
// 				pubsub.WithDirectPeers([]peer.AddrInfo{*host.InfoFromHost(cl.hs[id1]), *host.InfoFromHost(cl.hs[id2])}))
// 			if err != nil {
// 				panic(err)
// 			}
// 			cl.cs[id], err = cluster.New(ctx, ps)
// 			if err != nil {
// 				panic(err)
// 			}

// 		}(i)
// 	}

// 	wg.Wait()
// 	return &cl
// }

// func (cl *Cluster) Close() {
// 	for i := 0; i < len(cl.cs); i++ {
// 		cl.cs[i].Close()
// 		cl.hs[i].Close()
// 	}
// }

// func clusterView(ctx context.Context, c *ClusterClient, bufSize int32) (ps peer.IDSlice) {
// 	for it := c.Iter(bufSize); it.Record(ctx) != nil; it.Next(ctx) {
// 		ps = append(ps, it.Record(ctx).Peer())
// 	}

// 	return
// }
