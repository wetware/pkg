package routing

import (
	"context"
	"time"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	cluster "github.com/wetware/casm/pkg/cluster/routing"
	api "github.com/wetware/ww/internal/api/routing"
)

type iteration struct {
	rec      cluster.Record
	deadline int64
}

func newIteration(capIt api.Iteration) iteration {
	rec, _ := capIt.Record()
	return iteration{rec: newRecord(rec), deadline: capIt.Dedadline()}
}

func newIterations(capIts api.Iteration_List) []iteration {
	its := make([]iteration, 0, capIts.Len())
	for i := 0; i < capIts.Len(); i++ {
		its = append(its, newIteration(capIts.At(i)))
	}
	return its
}

type handler struct {
	ms      chan []iteration
	release capnp.ReleaseFunc
	ctx     context.Context
}

func (h handler) Shutdown() {
	close(h.ms)
	h.release()
}

func (h handler) Handle(ctx context.Context, call api.Routing_Handler_handle) error {
	iterations, err := call.Args().Iterations()
	if err != nil {
		return err
	}

	select {
	case h.ms <- newIterations(iterations):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-h.ctx.Done():
		return h.ctx.Err()
	}
}

type Iterator struct {
	h handler

	f      *capnp.Future
	cancel context.CancelFunc
	it     []iteration
	i      int
}

func newIterator(ctx context.Context, r api.Routing, bufSize int32) *Iterator {
	ctx, cancel := context.WithCancel(ctx)

	h := handler{
		ms:      make(chan []iteration),
		release: r.AddRef().Release,
		ctx:     ctx,
	}
	c := api.Routing_Handler_ServerToClient(h, &server.Policy{
		MaxConcurrentCalls: int(bufSize),
		AnswerQueueSize:    int(bufSize),
	})
	defer c.Release()

	f, release := r.Iter(
		ctx,
		func(ps api.Routing_iter_Params) error {
			ps.SetBufSize(bufSize)
			return ps.SetHandler(c.AddRef())
		})
	defer release()

	select {
	case <-ctx.Done():
		return nil
	case <-f.Done():
		if _, err := f.Struct(); err != nil {
			return nil
		}
	}

	return &Iterator{
		h:      h,
		f:      f.Future,
		cancel: cancel,
		i:      -1,
	}
}

func (it *Iterator) Next(ctx context.Context) {
	if len(it.it) > 0 && it.i+1 < len(it.it) {
		it.i++
		return
	}

	it.i = 0
	it.it = nil

	select {
	case iteration, ok := <-it.h.ms:
		if ok {
			it.it = iteration
		}
		return
	case <-ctx.Done():
		return
	}
}

func (it *Iterator) Record(ctx context.Context) cluster.Record {
	if it.isFirstCall() {
		it.Next(ctx)
	}

	if it.it == nil {
		return nil
	}
	return it.it[it.i].rec
}

func (it *Iterator) Deadline() time.Time {
	if it.isFirstCall() {
		it.Next(context.Background())
	}
	return time.UnixMicro(it.it[it.i].deadline)
}

func (it *Iterator) Finish() {
	it.cancel()
}

func (it *Iterator) isFirstCall() bool {
	return it.i == -1
}

type subHandler struct {
	handler api.Routing_Handler
	bufSize int32
}

func (sh subHandler) Handle(ctx context.Context, it cluster.Iterator) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for {
		if it.Record() == nil {
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

func (sh subHandler) send(ctx context.Context, it cluster.Iterator, abort func()) {
	recs := make([]cluster.Record, 0, sh.bufSize)
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
	// Abort the subscription if we receive a 'call on released client' exception.
	// This signals that the remote end has canceled their subscription.
	//
	// TODO:  test specifically for 'capnp: call on released client'.
	if _, err := f.Struct(); err != nil {
		abort()
	}
}
