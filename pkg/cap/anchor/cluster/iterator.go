package cluster

import (
	"context"
	"errors"
	"time"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	cluster "github.com/wetware/casm/pkg/cluster/routing"
	api "github.com/wetware/ww/internal/api/cluster"
)

var ErrExhausted = errors.New("exhausted")

type handler struct {
	ms      chan []cluster.Record
	release capnp.ReleaseFunc
}

func (h handler) Shutdown() {
	close(h.ms)
	h.release()
}

func (h handler) Handle(_ context.Context, call api.Cluster_Handler_handle) error {
	capRecs, err := call.Args().Records()
	if err != nil {
		return err
	}

	// Defensive programming.  Zero-length record slice causes Iterator.Next()
	// to panic.
	if capRecs.Len() == 0 {
		return nil
	}

	recs, err := newRecords(time.Now(), capRecs)
	if err == nil {
		h.ms <- recs // buffered
	}

	return err
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

	curr cluster.Record
	recs []cluster.Record
}

func newIterator(r api.Cluster, bufSize, lim uint8) *Iterator {

	h := handler{
		ms:      make(chan []cluster.Record, lim),
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
			ps.SetBufSize(lim)
			return ps.SetHandler(c.AddRef())
		})

	return &Iterator{
		h: h,

		fut: f.Future,
		release: func() {
			cancel()
			release()
		},
	}
}

func (it *Iterator) Next(ctx context.Context) error {
	if len(it.recs) == 0 {
		if err := it.nextBatch(ctx); err != nil {
			return err
		}
	}

	it.curr, it.recs = it.recs[0], it.recs[1:]
	return nil
}

func (it *Iterator) Record() cluster.Record {
	return it.curr
}

func (it *Iterator) Deadline() time.Time {
	if it.curr == nil {
		return time.Time{}
	}

	return time.Now().Add(it.curr.TTL())
}

func (it *Iterator) Finish() {
	if it.recs != nil && it.curr != nil {
		it.curr = nil
		it.recs = nil
		it.release()
	}
}

func (it *Iterator) nextBatch(ctx context.Context) (err error) {
	select {
	case recs, ok := <-it.h.ms:
		if ok {
			it.recs = recs
			return
		}

		err = ErrExhausted
		defer it.Finish()

	case <-ctx.Done():
		err = ctx.Err()
		defer it.Finish()
	}

	select {
	case <-it.fut.Done():
		_, err = it.fut.Struct()
		defer it.release()

	case <-ctx.Done():
		err = ctx.Err()
		defer it.Finish()
	}

	return
}
