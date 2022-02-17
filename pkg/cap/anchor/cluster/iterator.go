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

type handler chan []cluster.Record

func (h handler) Shutdown() { close(h) }

func (h handler) Handle(ctx context.Context, call api.Cluster_Handler_handle) error {
	capRecs, err := call.Args().Records()
	if err != nil || capRecs.Len() == 0 { // defensive
		return err
	}

	recs, err := newRecords(time.Now(), capRecs)
	if err != nil {
		return err
	}

	select {
	case h <- recs:
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

	fut *capnp.Future

	curr cluster.Record
	recs []cluster.Record
}

func newIterator(ctx context.Context, r api.Cluster, h handler) (*Iterator, capnp.ReleaseFunc) {
	c := api.Cluster_Handler_ServerToClient(h, &server.Policy{
		MaxConcurrentCalls: cap(h),
		AnswerQueueSize:    cap(h),
	})

	f, release := r.Iter(ctx, func(ps api.Cluster_iter_Params) error {
		return ps.SetHandler(c)
	})

	it := &Iterator{
		h:   h,
		fut: f.Future,
	}

	return it, func() {
		if it.recs != nil && it.curr != nil {
			it.curr = nil
			it.recs = nil
			release()
		}
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

func (it *Iterator) nextBatch(ctx context.Context) error {
	var ok bool
	select {
	case it.recs, ok = <-it.h:
		if ok {
			return nil
		}

		select {
		case <-it.fut.Done():
			if _, err := it.fut.Struct(); err != nil {
				return err
			}
			return ErrExhausted

		case <-ctx.Done():
		}

	case <-ctx.Done():
	}

	return ctx.Err()
}
