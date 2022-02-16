package cluster

import (
	"context"
	"errors"
	"fmt"
	"time"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	cluster "github.com/wetware/casm/pkg/cluster/routing"
	api "github.com/wetware/ww/internal/api/cluster"
)

var ErrClosed = errors.New("closed")

type handler struct {
	ms      chan []cluster.Record
	release capnp.ReleaseFunc
}

func (h handler) Shutdown() {
	close(h.ms)
	h.release()
}

func (h handler) Handle(ctx context.Context, call api.Cluster_Handler_handle) error {
	capRecs, err := call.Args().Records()
	if err != nil {
		return err
	}

	recs, err := newRecords(time.Now(), capRecs)
	if err != nil {
		return err
	}

	select {
	case h.ms <- recs:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func newRecords(t time.Time, capRecs api.Cluster_Record_List) ([]cluster.Record, error) {
	recs := make([]cluster.Record, 0, capRecs.Len())
	for i := 0; i < capRecs.Len(); i++ {
		rec, err := newRecord(t, capRecs.At(i))
		if err != nil {
			return nil, err
		}
		recs = append(recs, rec)
	}
	return recs, nil
}

type Iterator struct {
	h handler

	fut     *capnp.Future
	release capnp.ReleaseFunc

	recs []cluster.Record
	i    int

	finished bool
}

func newIterator(r api.Cluster, bufSize int32) *Iterator {
	h := handler{
		ms:      make(chan []cluster.Record),
		release: r.AddRef().Release,
	}
	c := api.Cluster_Handler_ServerToClient(h, &server.Policy{
		MaxConcurrentCalls: cap(h.ms),
		AnswerQueueSize:    cap(h.ms),
	})
	defer c.Release()

	ctx, cancel := context.WithCancel(context.Background())

	f, release := r.Iter(
		ctx,
		func(ps api.Cluster_iter_Params) error {
			ps.SetBufSize(bufSize)
			return ps.SetHandler(c.AddRef())
		})

	return &Iterator{
		h: h,

		fut: f.Future,
		release: func() {
			cancel()
			release()
		},

		recs: nil,
		i:    -1,

		finished: false,
	}
}

func (it *Iterator) Next(ctx context.Context) error {
	if it.finished {
		return ErrClosed
	}

	if it.recs != nil && len(it.recs) > 0 && it.i+1 < len(it.recs) {
		it.i++
		return nil
	}

	var err error

	select {
	case iteration, ok := <-it.h.ms:
		if ok {
			it.i = 0
			it.recs = iteration
			return nil
		}
		err = fmt.Errorf("%s unexpectedly", ErrClosed)
	case <-it.fut.Done():
		_, err = it.fut.Struct()
	case <-ctx.Done():
		err = ctx.Err()
	}
	it.Finish()
	return err
}

func (it *Iterator) Record(ctx context.Context) cluster.Record {
	if it.isFirstCall() {
		it.Next(ctx)
	}

	if it.finished || len(it.recs) == 0 {
		return nil
	}
	return it.recs[it.i]
}

func (it *Iterator) Deadline(ctx context.Context) time.Time {
	if it.isFirstCall() {
		it.Next(ctx)
	}

	if it.finished || len(it.recs) == 0 {
		return time.UnixMicro(0)
	}
	return time.Now().Add(it.recs[it.i].TTL())
}

func (it *Iterator) Finish() {
	if !it.finished {
		it.finished = true
		it.recs = nil
		it.release()
	}
}

func (it *Iterator) isFirstCall() bool {
	return it.i == -1
}
