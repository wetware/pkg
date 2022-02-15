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
	node *cluster.Node
	ctx  context.Context
}

func (rs *ClusterServer) NewClient(policy *server.Policy) ClusterClient {
	return ClusterClient{api.Cluster_ServerToClient(rs, policy)}
}

func (rs *ClusterServer) Iter(ctx context.Context, call api.Cluster_iter) error {
	h := serverIterator{
		handler: call.Args().Handler(),
		bufSize: call.Args().BufSize(),
	}
	defer h.handler.Release()

	it := rs.node.View().Iter()
	defer it.Finish()

	call.Ack()

	h.ServeHandler(rs.ctx, it)

	return nil
}

func (rs *ClusterServer) Lookup(_ context.Context, call api.Cluster_lookup) error {
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

type serverIterator struct {
	handler api.Cluster_Handler
	bufSize int32
}

func (sh serverIterator) ServeHandler(ctx context.Context, it routing.Iterator) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		if it.Record() == nil {
			sh.send(ctx, it, cancel) // send an empty iteration as a signal
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		sh.send(ctx, it, cancel)
	}
}

func (sh serverIterator) send(ctx context.Context, it routing.Iterator, abort func()) {
	recs := make([]routing.Record, 0, sh.bufSize)
	deadlines := make([]time.Time, 0, sh.bufSize)
	for i := 0; i < int(sh.bufSize) && it.Record() != nil; i++ {
		recs = append(recs, it.Record())
		deadlines = append(deadlines, it.Deadline())
		it.Next()
	}

	f, release := sh.handler.Handle(ctx,
		func(ps api.Cluster_Handler_handle_Params) error {
			its, err := ps.NewIterations(int32(len(recs)))
			if err != nil {
				abort()
			}
			for i := 0; i < len(recs); i++ {
				rec, err := its.At(i).NewRecord()
				if err != nil {
					abort()
				}
				rec.SetPeer(string(recs[i].Peer()))
				rec.SetSeq(recs[i].Seq())
				rec.SetTtl(int64(recs[i].TTL()))

				its.At(i).SetDeadline(deadlines[i].UnixMicro())
			}
			return nil
		})
	defer release()

	select {
	case <-f.Done():
	case <-ctx.Done():
		return
	}

	if _, err := f.Struct(); err != nil {
		abort()
	}
}
