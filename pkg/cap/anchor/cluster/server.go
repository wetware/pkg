package cluster

import (
	"context"
	"time"

	"capnproto.org/go/capnp/v3/server"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/casm/pkg/cluster/routing"
	api "github.com/wetware/ww/internal/api/cluster"
)

var defaultPolicy = server.Policy{
	// HACK:  raise MaxConcurrentCalls to mitigate known deadlock condition.
	//        https://github.com/capnproto/go-capnproto2/issues/189
	MaxConcurrentCalls: 64,
	AnswerQueueSize:    64,
}

type ClusterServer struct {
	view cluster.View
	ctx  context.Context
}

func (cs *ClusterServer) NewClient(policy *server.Policy) Client {
	return Client(api.Cluster_ServerToClient(cs, policy))
}

func (cs *ClusterServer) Iter(ctx context.Context, call api.Cluster_iter) error {
	call.Ack()
	return cs.serveHandler(ctx, call.Args().Handler(), call.Args().BufSize())
}

func (cs *ClusterServer) Lookup(_ context.Context, call api.Cluster_lookup) error {
	peerID, err := call.Args().PeerID()
	if err != nil {
		return err
	}
	capRec, ok := cs.view.Lookup(peer.ID(peerID))
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

type serverIterator struct {
	it      routing.Iterator
	handler api.Cluster_Handler
	bufSize int32
}

func (cs *ClusterServer) serveHandler(ctx context.Context, handler api.Cluster_Handler, bufSize int32) error {
	sit := serverIterator{cs.view.Iter(), handler, bufSize}
	defer sit.it.Finish()

	for {
		if sit.it.Record() == nil {
			cs.send(ctx, sit) // send an empty iteration as a signal
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-cs.ctx.Done():
			return cs.ctx.Err()
		default:
			err := cs.send(ctx, sit)
			if err != nil {
				return err
			}
		}
	}
}

func (cs *ClusterServer) send(ctx context.Context, sit serverIterator) error {
	recs := make([]routing.Record, 0, sit.bufSize)
	deadlines := make([]time.Time, 0, sit.bufSize)
	for i := 0; i < int(sit.bufSize) && sit.it.Record() != nil; i++ {
		recs = append(recs, sit.it.Record())
		deadlines = append(deadlines, sit.it.Deadline())
		sit.it.Next()
	}

	now := time.Now()

	f, release := sit.handler.Handle(ctx,
		func(ps api.Cluster_Handler_handle_Params) error {
			its, err := ps.NewRecords(int32(len(recs)))
			if err != nil {
				return err
			}
			for i := 0; i < len(recs); i++ {
				its.At(i).SetPeer(string(recs[i].Peer()))
				its.At(i).SetSeq(recs[i].Seq())
				its.At(i).SetTtl(deadlines[i].Sub(now).Microseconds())
			}
			return nil
		})
	defer release()

	select {
	case <-f.Done():
		_, err := f.Struct()
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-cs.ctx.Done():
		return cs.ctx.Err()
	}
}
