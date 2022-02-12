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
	"github.com/wetware/casm/pkg/cluster"
	mx "github.com/wetware/matrix/pkg"
)

var (
	nodesAmount = 10
	hs          = make([]host.Host, nodesAmount)
	cs          = make([]*cluster.Node, nodesAmount)
	tick        = 100 * time.Millisecond
	waitFor     = 10 * time.Second
)

func TestRoutingIter(t *testing.T){
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sim := mx.New(ctx)
	initCluster(ctx, sim)

	s := RoutingServer{cs[0]}
	c := s.NewClient(nil)

	time.Sleep(5*time.Second)
	fut, release := c.Iter(ctx)
	defer release()
	it := fut.Iterator()
	it.Next(ctx, 1)
	recs, err := it.Records(ctx)
	if err!=nil{
		t.Error(err)
	}
	println(len(recs), recs[0])
	println(len(recs), recs[0].Peer())


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

/* func clusterView(ctx context.Context, c *RoutingClient) (ps peer.IDSlice) {
	fut, release := c.Iter(ctx)
	defer release()

	it := fut.Iterator()
	println("Records len:", len(it.Records(ctx)))
	
	for ;len(it.Records(ctx))>0; it.Next(ctx, 1) {
		ps = append(ps, it.Records(ctx)[0].Peer())
		println("Records len:", len(it.Records(ctx)))
	}
	return
} */