package routing

import (
	"context"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/wetware/casm/pkg/cluster"
	mx "github.com/wetware/matrix/pkg"
)

var (
	nodesAmount = 5
)

type Cluster struct {
	hs []host.Host
	cs []*cluster.Node
}

func TestRoutingIter(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sim := mx.New(ctx)
	cl := newCluster(ctx, sim)
	defer cl.Close()

	s := RoutingServer{cl.cs[0], ctx}
	c := s.NewClient(nil)

	assert.Eventually(t,
		func() bool {
			return len(clusterView(ctx, &c, 2)) == nodesAmount
		},
		time.Second*5,
		time.Millisecond*10,
		"peers should receive each other's bootstrap messages")
}

func TestRoutingIterClosedUnexpected(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sim := mx.New(ctx)
	cl := newCluster(ctx, sim)
	defer cl.Close()

	assert.Eventually(t,
		func() bool {
			ctx1, cancel1 := context.WithCancel(ctx)
			ctx2, cancel2 := context.WithCancel(ctx)
			defer cancel2()

			s := RoutingServer{cl.cs[0], ctx1}
			c := s.NewClient(nil)

			it := c.Iter(1)
			cancel1()
			return it.Next(ctx2) == ErrClosedUnexpected
		},
		time.Second*5,
		time.Millisecond*10,
		"when the server fails, the iterator should raise an error")
}

func TestRoutingLookup(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cl := newCluster(ctx, mx.New(ctx))
	defer cl.Close()

	s := RoutingServer{cl.cs[0], ctx}
	c := s.NewClient(nil)

	id := cl.hs[rand.Intn(nodesAmount)].ID()
	assert.Eventually(t,
		func() bool {
			rec, err := c.Lookup(ctx, id)
			return err == nil && rec.Peer() == peer.ID(id.String())
		},
		time.Second*8,
		time.Millisecond*10,
		"peers should receive each other's bootstrap messages")

}

func newCluster(ctx context.Context, sim mx.Simulation) *Cluster {
	var (
		cl = Cluster{
			hs: make([]host.Host, nodesAmount),
			cs: make([]*cluster.Node, nodesAmount),
		}
		wg sync.WaitGroup
	)

	// init hosts
	for i := 0; i < nodesAmount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			cl.hs[id] = sim.MustHost(ctx)
		}(i)
	}
	wg.Wait()

	// init cluster nodes
	for i := 0; i < nodesAmount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// decide neighbors to bootstrap pubsub
			id1, id2 := id-1, id+1
			if id1 < 0 {
				id1 = nodesAmount - 1
			}
			if id2 >= nodesAmount {
				id2 = 0
			}

			// init pubsub + cluster node
			ps, err := pubsub.NewGossipSub(ctx, cl.hs[id],
				pubsub.WithDirectPeers([]peer.AddrInfo{*host.InfoFromHost(cl.hs[id1]), *host.InfoFromHost(cl.hs[id2])}))
			if err != nil {
				panic(err)
			}
			cl.cs[id], err = cluster.New(ctx, ps)
			if err != nil {
				panic(err)
			}

		}(i)
	}

	wg.Wait()
	return &cl
}

func (cl *Cluster) Close() {
	for i := 0; i < len(cl.cs); i++ {
		cl.cs[i].Close()
		cl.hs[i].Close()
	}
}

func clusterView(ctx context.Context, c *RoutingClient, bufSize int32) (ps peer.IDSlice) {
	it := c.Iter(bufSize)
	defer it.Finish()

	for ; it.Record(ctx) != nil; it.Next(ctx) {
		ps = append(ps, it.Record(ctx).Peer())
	}

	return
}
