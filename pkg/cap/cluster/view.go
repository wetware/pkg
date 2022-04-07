package cluster

import (
	"context"
	"time"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wetware/casm/pkg/cluster/routing"
	api "github.com/wetware/ww/internal/api/cluster"
	"github.com/wetware/ww/pkg/vat"
	"golang.org/x/sync/semaphore"
)

var (
	ViewCapability = vat.BasicCap{
		"view/packed",
		"view"}
)

const (
	defaultBatchSize   = 64
	defaultMaxInflight = 8
)

var defaultPolicy = server.Policy{
	// HACK:  raise MaxConcurrentCalls to mitigate known deadlock condition.
	//        https://github.com/capnproto/go-capnproto2/issues/189
	MaxConcurrentCalls: 64,
	AnswerQueueSize:    64,
}

// RoutingTable provides a global view of namespace peers.
type RoutingTable interface {
	Iter() routing.Iterator
	Lookup(peer.ID) (routing.Record, bool)
}

type ViewServer struct {
	View RoutingTable
}

func NewViewServer(rt RoutingTable) ViewServer {
	vs := ViewServer{View: rt}

	return vs
}

func (f ViewServer) NewClient(policy *server.Policy) View {
	if policy == nil {
		policy = &defaultPolicy
	}

	return View(api.View_ServerToClient(f, policy))
}

func (f ViewServer) Client() *capnp.Client {
	return api.View_ServerToClient(f, &defaultPolicy).Client
}

func (f ViewServer) Iter(ctx context.Context, call api.View_iter) error {
	call.Ack()

	b := newBatcher(call.Args())

	for it := f.View.Iter(); it.Record() != nil; it.Next() {
		if err := b.Send(ctx, it.Record(), it.Deadline()); err != nil {
			it.Finish()
			return err
		}
	}

	return b.Wait(ctx)
}

func (f ViewServer) Lookup(_ context.Context, call api.View_lookup) error {
	peerID, err := call.Args().PeerID()
	if err != nil {
		return err
	}
	capRec, ok := f.View.Lookup(peer.ID(peerID))
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
		rec.SetPeer(string(capRec.Peer()))
		rec.SetTtl(int64(capRec.TTL()))
		rec.SetSeq(capRec.Seq())
	}
	return nil
}

type View api.View

func (v View) Iter(ctx context.Context) (*RecordStream, capnp.ReleaseFunc) {
	ctx, cancel := context.WithCancel(ctx)

	h := make(handler, defaultMaxInflight)

	it, release := newIterator(ctx, api.View(v), h)
	return it, func() {
		cancel()
		release()
	}
}

func (v View) Lookup(ctx context.Context, peerID peer.ID) (routing.Record, error) {
	f, release := api.View(v).Lookup(ctx, func(r api.View_lookup_Params) error {
		return r.SetPeerID(string(peerID))
	})
	defer release()

	res, err := f.Struct()
	if err != nil {
		return nil, err
	}

	r, err := res.Record()
	if err != nil {
		return nil, err
	}

	return recordFromCapnp(r)
}

type Record api.View_Record

func (rec Record) TTL() time.Duration {
	return time.Duration(api.View_Record(rec).Ttl())
}

func (rec Record) Seq() uint64 {
	return api.View_Record(rec).Seq()
}

type RecordStream struct {
	h <-chan []record
	f *capnp.Future

	Err error

	head record
	tail []record
}

func newIterator(ctx context.Context, r api.View, h handler) (*RecordStream, capnp.ReleaseFunc) {
	c := api.View_Handler_ServerToClient(h, &server.Policy{
		MaxConcurrentCalls: cap(h),
		AnswerQueueSize:    cap(h),
	})

	f, release := r.Iter(ctx, func(ps api.View_iter_Params) error {
		return ps.SetHandler(c)
	})

	return &RecordStream{h: h, f: f.Future}, release
}

func (it *RecordStream) Next(ctx context.Context) (more bool) {
	if len(it.tail) == 0 {
		it.Err = it.nextBatch(ctx)
	}

	if more = it.Err == nil && len(it.tail) > 0; more {
		it.head, it.tail = it.tail[0], it.tail[1:]
	}

	return
}

func (it *RecordStream) Record() routing.Record { return it.head }

func (it *RecordStream) nextBatch(ctx context.Context) (err error) {
	var ok bool
	select {
	case it.tail, ok = <-it.h:
		if !ok {
			_, err = it.f.Struct()
		}

	case <-ctx.Done():
		err = ctx.Err()
	}

	return
}

type handler chan []record

func (h handler) Shutdown() { close(h) }

func (h handler) Handle(ctx context.Context, call api.View_Handler_handle) error {
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

func loadBatch(args api.View_Handler_handle_Params) ([]record, error) {
	rs, err := args.Records()
	if err != nil {
		return nil, err
	}

	batch := make([]record, rs.Len())
	for i := range batch {
		batch[i], err = recordFromCapnp(rs.At(i))
		if err != nil {
			break
		}
	}

	return batch, err
}

type record struct {
	id  peer.ID
	ttl time.Duration
	seq uint64
}

func recordFromCapnp(r api.View_Record) (rec record, err error) {
	var s string
	if s, err = r.Peer(); err == nil {
		rec.seq = r.Seq()
		rec.ttl = time.Duration(r.Ttl())
		rec.id, err = peer.IDFromString(s)
	}

	return
}

func (r record) Peer() peer.ID      { return r.id }
func (r record) TTL() time.Duration { return r.ttl }
func (r record) Seq() uint64        { return r.seq }

type batcher struct {
	lim   *limiter
	h     api.View_Handler
	fs    map[*capnp.Future]capnp.ReleaseFunc // in-flight
	batch batch
}

func newBatcher(p api.View_iter_Params) batcher {
	return batcher{
		lim:   newLimiter(),
		h:     p.Handler(),
		fs:    make(map[*capnp.Future]capnp.ReleaseFunc),
		batch: newBatch(),
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

func newBatch() batch {
	return batch{
		rs: make([]batchRecord, 0, defaultBatchSize),
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

func (b *batch) Flush() func(api.View_Handler_handle_Params) error {
	return func(p api.View_Handler_handle_Params) error {
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
		if b.t.Before(r.Deadline) {
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

func (r batchRecord) SetParam(t time.Time, rec api.View_Record) error {
	rec.SetSeq(r.Seq)
	rec.SetTtl(r.Deadline.Sub(t).Microseconds())
	return rec.SetPeer(string(r.ID))
}

type limiter semaphore.Weighted

func newLimiter() *limiter {
	return (*limiter)(semaphore.NewWeighted(defaultMaxInflight))
}

func (l *limiter) Acquire(ctx context.Context) error {
	return (*semaphore.Weighted)(l).Acquire(ctx, 1)
}

func (l *limiter) Release() {
	(*semaphore.Weighted)(l).Release(1)
}
