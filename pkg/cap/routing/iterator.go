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
}

func (h handler) Shutdown() {
	println("Shutdown")
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
	}
}

type IteratorV2 struct {
	h handler

	err     error // future resolution error
	f       *capnp.Future
	release capnp.ReleaseFunc
	it      []iteration
	i       int
}

func newIterator(ctx context.Context, r api.Routing, bufSize int32) *IteratorV2 {
	h := handler{
		ms:      make(chan []iteration),
		release: r.AddRef().Release,
	}
	c := api.Routing_Handler_ServerToClient(h, &server.Policy{
		MaxConcurrentCalls: int(bufSize),
		AnswerQueueSize:    int(bufSize),
	})
	defer c.Release()

	f, release := r.Iter(
		context.Background(),
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

	return &IteratorV2{
		h:       h,
		f:       f.Future,
		release: c.AddRef().Release,
		i:       -1,
	}
}

func (it *IteratorV2) Next(ctx context.Context) {
	if err := it.Resolve(ctx); err != nil {
		return
	}

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

func (it *IteratorV2) Record(ctx context.Context) cluster.Record {
	if it.isFirstCall() {
		it.Next(ctx)
	}

	if it.it == nil {
		return nil
	}
	return it.it[it.i].rec
}

func (it *IteratorV2) Deadline() time.Time {
	if it.isFirstCall() {
		it.Next(context.Background())
	}
	return time.UnixMicro(it.it[it.i].deadline)
}

func (it *IteratorV2) Finish() {
	//TODO: clean up?
}

func (it *IteratorV2) isFirstCall() bool {
	return it.i == -1
}

// Resolve blocks until the subscription is ready, the underlying
// RPC call fails, or the context expires. If the RPC call fails,
// the subscription is automatically canceled.
func (it *IteratorV2) Resolve(ctx context.Context) error {
	if it.release != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-it.f.Done():
			_, it.err = it.f.Struct()
			it.release()

			// free memory
			it.release = nil
			it.f = nil
		}
	}

	return it.err
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
		sh.send(ctx, it, cancel)
		it.Next()
	}
}

func (sh subHandler) send(ctx context.Context, it cluster.Iterator, abort func()) {
	f, release := sh.handler.Handle(ctx,
		func(ps api.Routing_Handler_handle_Params) error {
			its, err := ps.NewIterations(int32(sh.bufSize))
			if err != nil {
				abort()
			}
			for i := 0; i < int(sh.bufSize); i++ {
				rec, err := its.At(i).NewRecord()
				if err != nil {
					abort()
				}
				rec.SetPeer(string(it.Record().Peer()))
				rec.SetSeq(it.Record().Seq())
				rec.SetTtl(int64(it.Record().TTL()))

				its.At(i).SetDedadline(it.Deadline().UnixMicro())
			}
			return nil
		})
	defer release()

	// Abort the subscription if we receive a 'call on released client' exception.
	// This signals that the remote end has canceled their subscription.
	//
	// TODO:  test specifically for 'capnp: call on released client'.
	if _, err := f.Struct(); err != nil {
		abort()
	}
}
