package cluster

import (
	"context"
	"errors"
	"time"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	"github.com/libp2p/go-libp2p-core/peer"
	cluster "github.com/wetware/casm/pkg/cluster/routing"
	api "github.com/wetware/ww/internal/api/cluster"
)

var ErrExhausted = errors.New("exhausted")

type handler chan []record

func (h handler) Shutdown() { close(h) }

func (h handler) Handle(ctx context.Context, call api.Cluster_Handler_handle) error {
	recs, err := loadBatch(call.Args())
	if err != nil || len(recs) == 0 { // defensive
		return err
	}

	select {
	case h <- recs:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

func loadBatch(args api.Cluster_Handler_handle_Params) ([]record, error) {
	rs, err := args.Records()
	if err != nil {
		return nil, err
	}

	batch := make([]record, rs.Len())
	for i := range batch {
		rec := Record(rs.At(i))

		batch[i].ttl = rec.TTL()
		batch[i].seq = rec.Seq()

		if batch[i].id, err = rec.Peer(); err != nil {
			break
		}
	}

	return batch, nil
}

type record struct {
	id  peer.ID
	ttl time.Duration
	seq uint64
}

func (r record) Peer() peer.ID      { return r.id }
func (r record) TTL() time.Duration { return r.ttl }
func (r record) Seq() uint64        { return r.seq }

type Iterator struct {
	h handler

	Err error
	fut *capnp.Future

	curr cluster.Record
	recs []record
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
		if it.Err = it.nextBatch(ctx); it.Err != nil {
			return it.Err
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
