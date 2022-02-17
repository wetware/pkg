package cluster

import (
	"context"
	"time"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/casm/pkg/cluster/routing"
	api "github.com/wetware/ww/internal/api/cluster"
	"golang.org/x/sync/semaphore"
)

var defaultPolicy = server.Policy{
	// HACK:  raise MaxConcurrentCalls to mitigate known deadlock condition.
	//        https://github.com/capnproto/go-capnproto2/issues/189
	MaxConcurrentCalls: 64,
	AnswerQueueSize:    64,
}

type ClusterServer struct{ cluster.View }

func (cs *ClusterServer) NewClient(policy *server.Policy) Client {
	if policy == nil {
		policy = &defaultPolicy
	}

	return Client(api.Cluster_ServerToClient(cs, policy))
}

func (cs *ClusterServer) Iter(ctx context.Context, call api.Cluster_iter) error {
	call.Ack()

	b := newBatcher(call.Args())

	for it := cs.View.Iter(); it.Record() != nil; it.Next() {
		if err := b.Send(ctx, it.Record(), it.Deadline()); err != nil {
			it.Finish()
			return err
		}
	}

	return b.Wait(ctx)
}

func (cs *ClusterServer) Lookup(_ context.Context, call api.Cluster_lookup) error {
	peerID, err := call.Args().PeerID()
	if err != nil {
		return err
	}
	capRec, ok := cs.View.Lookup(peer.ID(peerID))
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

type batcher struct {
	lim   *limiter
	h     api.Cluster_Handler
	fs    map[*capnp.Future]capnp.ReleaseFunc // in-flight
	batch batch
}

func newBatcher(p api.Cluster_iter_Params) batcher {
	return batcher{
		lim:   newLimiter(p.Lim()),
		h:     p.Handler(),
		fs:    make(map[*capnp.Future]capnp.ReleaseFunc),
		batch: newBatch(p.BufSize()),
	}
}

func (b *batcher) Send(ctx context.Context, r routing.Record, dl time.Time) error {
	// batch is full?
	if b.batch.Add(r, dl) {
		return b.Flush(ctx, false)
	}

	return nil
}

func (b *batcher) Flush(ctx context.Context, force bool) error {
	if b.batch.FilterExpired() || force {
		if err := b.lim.Acquire(ctx); err != nil {
			return err
		}

		f, release := b.h.Handle(ctx, b.batch.Flush())
		b.fs[f.Future] = func() {
			delete(b.fs, f.Future)
			release()
			b.lim.Release()
		}
	}

	// release any resolved futures and return their errors, if any
	for f, release := range b.fs {
		select {
		case <-f.Done():
			defer release()
			if _, err := f.Struct(); err != nil {
				return err
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func (b *batcher) Wait(ctx context.Context) (err error) {
	if err = b.Flush(ctx, true); err != nil {
		return
	}

	for f, release := range b.fs {
		// This is a rare case in which the use of 'defer' in
		// a loop is NOT a bug.
		//
		// We iterate over the whole map in order to schedule
		// a deferred call to 'release' for all pending RPC
		// calls.
		//
		// In principle, this is not necessary since resources
		// will be released when the handler for Iter returns.
		// We do it anyway to guard against bugs and/or changes
		// in the capnp API.
		defer release()

		// We want to abort early if any future encounters an
		// error, but as per the previous comment, we also want
		// to defer a call to 'release' for each future.
		if err == nil {
			// We're waiting until all futures resolve, so it's
			// okay to block on any given 'f'.
			select {
			case <-f.Done():
				_, err = f.Struct()

			case <-ctx.Done():
				err = ctx.Err()
			}
		}
	}

	return
}

type batch struct {
	t  time.Time
	rs []batchRecord
}

func newBatch(size uint8) batch {
	if size == 0 {
		size = 32
	}

	return batch{
		rs: make([]batchRecord, 0, size),
	}
}

func (b *batch) Full() bool { return len(b.rs) == cap(b.rs) }
func (b *batch) Len() int32 { return int32(len(b.rs)) }

func (b *batch) Add(r routing.Record, dl time.Time) bool {
	b.rs = append(b.rs, batchRecord{
		ID:       r.Peer(),
		Seq:      r.Seq(),
		Deadline: dl.Truncate(time.Millisecond),
	})

	return b.Full()
}

func (b *batch) Flush() func(api.Cluster_Handler_handle_Params) error {
	return func(p api.Cluster_Handler_handle_Params) error {
		defer func() {
			b.rs = b.rs[:0]
		}()

		rs, err := p.NewRecords(b.Len())
		if err != nil {
			return err
		}

		for i, r := range b.rs {
			if err = r.SetParam(b.t, rs.At(i)); err != nil {
				break
			}
		}

		return err
	}
}

func (b *batch) FilterExpired() bool {
	b.t = time.Now().Truncate(time.Millisecond)

	current := b.rs[:]
	b.rs = b.rs[:0]
	for _, r := range current {
		if r.Deadline.Before(b.t) {
			b.rs = append(b.rs, r)
		}
	}

	return b.Full()
}

type batchRecord struct {
	ID       peer.ID
	Seq      uint64
	Deadline time.Time
}

func (r batchRecord) SetParam(t time.Time, rec api.Cluster_Record) error {
	rec.SetSeq(r.Seq)
	rec.SetTtl(r.Deadline.Sub(t).Microseconds())
	return rec.SetPeer(string(r.ID))
}

type limiter semaphore.Weighted

func newLimiter(lim uint8) *limiter {
	if lim == 0 {
		lim = 16
	}

	return (*limiter)(semaphore.NewWeighted(int64(lim)))
}

func (l *limiter) Acquire(ctx context.Context) error {
	return (*semaphore.Weighted)(l).Acquire(ctx, 1)
}

func (l *limiter) Release() {
	(*semaphore.Weighted)(l).Release(1)
}
