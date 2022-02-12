package routing

import (
	"context"
	"sync"
	"testing"
	"time"

	//"github.com/stretchr/testify/assert"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	//"github.com/stretchr/testify/assert"
	"github.com/wetware/casm/pkg/cluster"
	mx "github.com/wetware/matrix/pkg"
)

var (
	nodesAmount = 10
	hs          = make([]host.Host, nodesAmount)
	cs          = make([]*cluster.Node, nodesAmount)
)

func TestRoutingIter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sim := mx.New(ctx)
	initCluster(ctx, sim)
	defer closeCluster()

	s := RoutingServer{cs[0], ctx}
	c := s.NewClient(nil)

	time.Sleep(5 * time.Second)

	println(len(clusterView(ctx, &c)))

	/* assert.Eventually(t,
	func() bool {
		return len(clusterView(ctx, &c)) == nodesAmount
	},
	time.Second*5,
	time.Millisecond*10,
	"peers should receive each other's bootstrap messages") */

}

func initCluster(ctx context.Context, sim mx.Simulation) {
	var wg sync.WaitGroup

	// init hosts
	for i := 0; i < nodesAmount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			hs[id] = sim.MustHost(ctx)
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
			ps, err := pubsub.NewGossipSub(ctx, hs[id],
				pubsub.WithDirectPeers([]peer.AddrInfo{*host.InfoFromHost(hs[id1]), *host.InfoFromHost(hs[id2])}))
			if err != nil {
				panic(err)
			}
			cs[id], err = cluster.New(ctx, ps)
			if err != nil {
				panic(err)
			}

		}(i)
	}

	wg.Wait()
}

func closeCluster() {
	for i := 0; i < len(cs); i++ {
		cs[i].Close()
		hs[i].Close()
	}
}

func clusterView(ctx context.Context, c *RoutingClient) (ps peer.IDSlice) {
	it := c.Iter(ctx)
	defer it.Finish()

	println("Cluster view")
	for ; it.Record(ctx) != nil; it.Next(ctx) {
		println(it.Record(ctx).Peer()[:5])
		ps = append(ps, it.Record(ctx).Peer())
	}

	return
}
