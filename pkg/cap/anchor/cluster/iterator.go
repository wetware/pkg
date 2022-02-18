package cluster

import (
	"context"
	"time"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	"github.com/libp2p/go-libp2p-core/peer"
	cluster "github.com/wetware/casm/pkg/cluster/routing"
	api "github.com/wetware/ww/internal/api/cluster"
)

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
	r   resolver

	head cluster.Record
	tail []record
}

func newIterator(ctx context.Context, r api.Cluster, h handler) (*Iterator, capnp.ReleaseFunc) {
	c := api.Cluster_Handler_ServerToClient(h, &server.Policy{
		MaxConcurrentCalls: cap(h),
		AnswerQueueSize:    cap(h),
	})

	f, release := r.Iter(ctx, func(ps api.Cluster_iter_Params) error {
		return ps.SetHandler(c)
	})

	return &Iterator{
		h: h,
		r: resolver(f),
	}, release
}

func (it *Iterator) Next(ctx context.Context) (more bool) {
	if len(it.tail) == 0 {
		it.Err = it.nextBatch(ctx)
	}

	if more = it.Err == nil && len(it.tail) > 0; more {
		it.head, it.tail = it.tail[0], it.tail[1:]
	}

	return
}

func (it *Iterator) Record() cluster.Record {
	return it.head
}

func (it *Iterator) nextBatch(ctx context.Context) (err error) {
	var ok bool
	select {
	case it.tail, ok = <-it.h:
		if !ok {
			err = it.r.Resolve(ctx)
		}

	case <-ctx.Done():
		err = ctx.Err()
	}

	return
}

type resolver api.Cluster_iter_Results_Future

func (r resolver) Resolve(ctx context.Context) error {
	select {
	case <-r.Done():
		_, err := r.Struct()
		return err

	case <-ctx.Done():
		return ctx.Err()
	}
}
