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
	ctx  context.Context
}

func (rs *RoutingServer) NewClient(policy *server.Policy) RoutingClient {
	return RoutingClient{api.Routing_ServerToClient(rs, policy)}
}

func (rs *RoutingServer) Iter(ctx context.Context, call api.Routing_iter) error {
	h := subHandler{
		handler: call.Args().Handler().AddRef(),
		bufSize: call.Args().BufSize(),
	}
	it := rs.node.View().Iter()

	go func() {
		defer h.handler.Release()
		defer it.Finish()

		h.Handle(rs.ctx, it)
	}()

	return nil
}

func (rs *RoutingServer) Lookup(_ context.Context, call api.Routing_lookup) error {
	peerID, err := call.Args().PeerID()
	if err != nil {
		return err
	}
	capRec, ok := rs.node.View().Lookup(peer.ID(peerID))
	results, err := call.AllocResults()
	if err != nil {
		return err
	}
	rec, err := results.NewRecord()
	if err != nil {
		return err
	}

	results.SetOk(ok)

	if ok {
		rec.SetPeer(capRec.Peer().String())
		rec.SetTtl(int64(capRec.TTL()))
		rec.SetSeq(capRec.Seq())
	}
	return nil
}
