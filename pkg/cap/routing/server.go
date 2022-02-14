package routing

import (
	"context"
	"time"

	"capnproto.org/go/capnp/v3/server"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/casm/pkg/cluster/routing"
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
	h := serverIterator{
		handler: call.Args().Handler().AddRef(),
		bufSize: call.Args().BufSize(),
	}
	it := rs.node.View().Iter()

	go func() {
		defer h.handler.Release()
		defer it.Finish()

		h.ServeHandler(rs.ctx, it)
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

type serverIterator struct {
	handler api.Routing_Handler
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
		func(ps api.Routing_Handler_handle_Params) error {
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

				its.At(i).SetDedadline(deadlines[i].UnixMicro())
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
