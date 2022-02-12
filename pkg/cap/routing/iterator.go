package routing

import (
	"context"
	"errors"
	"time"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	cluster "github.com/wetware/casm/pkg/cluster/routing"
	api "github.com/wetware/ww/internal/api/routing"
)

type handler struct {
	cq chan struct{}
	ms chan api.Iteration_List
}

func newHandler() handler {
	return handler{
		cq: make(chan struct{}),
		ms: make(chan api.Iteration_List),
	}
}

func (h handler) Shutdown() {
	select {
	case <-h.cq:
		return
	default:
		close(h.cq)
	}
}

func (h handler) Handle(ctx context.Context, call api.Routing_Handler_handle) error {
	iterations, err := call.Args().Iterations()
	if err != nil {
		return err
	}

	select {
	case h.ms <- iterations:
		return nil
	case <-h.cq:
		return errors.New("closed")
	case <-ctx.Done():
		return ctx.Err()
	}
}

type IteratorV2 struct {
	h handler

	err     error // future resolution error
	f       *capnp.Future
	release capnp.ReleaseFunc
	it      api.Iteration_List
	i       int
}

func newIterator(r api.Routing, bufSize int32) *IteratorV2 {
	var (
		h = newHandler()
		c = api.Routing_Handler_ServerToClient(h, &server.Policy{
			MaxConcurrentCalls: int(bufSize),
			AnswerQueueSize:    int(bufSize),
		})

		f, release = r.Iter(
			context.Background(),
			func(ps api.Routing_iter_Params) error {
				ps.SetBufSize(bufSize)
				return ps.SetHandler(c)
			})
	)

	return &IteratorV2{
		h:       h,
		f:       f.Future,
		release: release,
		i:       -1,
	}
}

func (it *IteratorV2) Cancel() {
	if it.release != nil {
		it.release()
	}

	it.h.Shutdown()
}

func (it *IteratorV2) Next(ctx context.Context) {
	if err := it.Resolve(ctx); err != nil {
		return
	}

	if it.it.Len() > 0 && it.i < it.it.Len() {
		it.i++
		return
	}

	it.i = 0

	select {
	case iteration := <-it.h.ms:
		it.it = iteration
		return
	case <-it.h.cq:
		return

	case <-ctx.Done():
		return
	}
}

func (it *IteratorV2) Record(ctx context.Context) cluster.Record {
	if it.isFirstCall() {
		it.Next(ctx)
	}

	if it.isFinished() {
		return nil
	}

	rec, err := it.it.At(it.i).Record()
	if err != nil {
		return nil
	}

	return newRecord(rec)
}

func (it *IteratorV2) Deadline() time.Time {
	if it.isFirstCall() {
		it.Next(context.Background())
	}
	return time.UnixMicro(it.it.At(it.i).Dedadline())
}

func (it *IteratorV2) Finish() {
	it.release()
}

func (it *IteratorV2) isFirstCall() bool {
	return it.i == -1
}

func (it *IteratorV2) isFinished() bool {
	select {
	case <-it.h.cq:
		return true
	default:
		return false
	}
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
	defer sh.handler.Release()
	defer it.Finish()
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
