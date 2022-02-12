package routing

import (
	"context"

	capnp "capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wetware/casm/pkg/cluster"
	cRouting "github.com/wetware/casm/pkg/cluster/routing"
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
	println("NewClient")
	return RoutingClient{api.Routing_ServerToClient(rs, policy)}
}

func (rs *RoutingServer) Iter(ctx context.Context, call api.Routing_iter) error {
	println("Iter")
	results, err := call.AllocResults()
	if err != nil {
		return err
	}
	println("Iter")
	err = results.SetIterator(api.Iterator_ServerToClient(&IteratorServer{it: rs.node.View().Iter()}, &defaultPolicy))
	if err!=nil{
		println("Iter error")
	}
	println("Set Iter")
	return err
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
	rec, err := capnpRecord(crec)
	if err != nil {
		return err
	}
	results.SetRecord(rec)
	results.SetOk(ok)
	return nil
}

func capnpRecord(crec cRouting.Record) (rec api.Record, err error) {
	_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return rec, err
	}
	rec, err = api.NewRootRecord(seg)
	if err != nil {
		return rec, err
	}
	rec.SetPeer(crec.Peer().String())
	rec.SetTtl(int64(crec.TTL()))
	rec.SetSeq(crec.Seq())
	return rec, nil
}

type IteratorServer struct {
	it   cRouting.Iterator
	recs []cRouting.Record
}

func (is *IteratorServer) Next(_ context.Context, call api.Iterator_next) error {
	amount := call.Args().Amount()
	recs := make([]cRouting.Record, 0, amount)
	for i := 0; i < int(amount); i++ {
		rec := is.it.Record()
		if rec !=nil{
			recs = append(recs, is.it.Record())
			is.it.Next()
		}
	}
	is.recs = recs

	return nil
}

func (is *IteratorServer) Records(_ context.Context, call api.Iterator_records) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	recs, err := res.NewRecords(int32(len(is.recs)))
	if err!=nil{
		return err
	}

	for i, rec := range is.recs {
		recs.At(i).SetTtl(int64(rec.TTL()))
		recs.At(i).SetSeq(rec.Seq())
		err := recs.At(i).SetPeer(rec.Peer().String())
		if err!=nil{
			return err
		}
	}
	return nil
}

func (is *IteratorServer) Deadline(_ context.Context, call api.Iterator_deadline) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	res.SetDeadline(is.it.Deadline().UnixMicro())
	return nil
}

func (is *IteratorServer) Finish(_ context.Context, call api.Iterator_finish) error {
	_, err := call.AllocResults()
	if err != nil {
		return err
	}
	is.it.Finish()
	return nil
}
