package routing

import (
	"context"

	"capnproto.org/go/capnp/v3/server"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wetware/casm/pkg/cluster"
	api "github.com/wetware/ww/internal/api/routing"
)

var defaultPolicy = server.Policy{
	// HACK:  raise MaxConcurrentCalls to mitigate known deadlock condition.
	//        https://github.com/capnproto/go-capnproto2/issues/189
	MaxConcurrentCalls: 64,
	AnswerQueueSize:    64,
}

type RoutingServer struct {
	node *cluster.Node
}

func (rs *RoutingServer) NewClient(policy *server.Policy) RoutingClient {
	return RoutingClient{api.Routing_ServerToClient(rs, policy)}
}

func (rs *RoutingServer) Iter(ctx context.Context, call api.Routing_iter) error {
	go subHandler{
		handler: call.Args().Handler().AddRef(),
		bufSize: call.Args().BufSize(),
	}.Handle(ctx, rs.node.View().Iter())
	return nil
}

func (rs *RoutingServer) Lookup(_ context.Context, call api.Routing_lookup) error {
	peerID, err := call.Args().PeerID()
	if err != nil {
		return err
	}
	crec, ok := rs.node.View().Lookup(peer.ID(peerID))
	results, err := call.AllocResults()
	if err != nil {
		return err
	}
	rec, err := results.NewRecord()
	if err != nil {
		return err
	}
	rec.SetPeer(crec.Peer().String())
	rec.SetTtl(int64(crec.TTL()))
	rec.SetSeq(crec.Seq())

	results.SetOk(ok)
	return nil
}
