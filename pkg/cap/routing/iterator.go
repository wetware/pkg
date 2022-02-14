package routing

import (
	"context"
	"time"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	cluster "github.com/wetware/casm/pkg/cluster/routing"
	api "github.com/wetware/ww/internal/api/routing"
)

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

func newIterator(r api.Routing, bufSize int32) *Iterator {
	ctx, cancel := context.WithCancel(context.Background())

	h := handler{
		ms:      make(chan []iteration),
		release: r.AddRef().Release,
		ctx:     ctx,
	}
	c := api.Routing_Handler_ServerToClient(h, &server.Policy{
		MaxConcurrentCalls: cap(h.ms),
		AnswerQueueSize:    cap(h.ms),
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
