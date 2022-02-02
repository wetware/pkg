package pubsub

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"
	ctxutil "github.com/lthibault/util/ctx"
	api "github.com/wetware/ww/internal/api/pubsub"
	"golang.org/x/sync/semaphore"
)

var ErrClosed = errors.New("closed")

var defaultPolicy = server.Policy{
	// HACK:  raise MaxConcurrentCalls to mitigate known deadlock condition.
	//        https://github.com/capnproto/go-capnproto2/issues/189
	MaxConcurrentCalls: 64,
	AnswerQueueSize:    64,
}

type TopicJoiner interface {
	Join(string, ...pubsub.TopicOpt) (*pubsub.Topic, error)
}

// Factory wraps a PubSub and provides a NewCap() factory method that
// returns a client capability for the pubsub.
//
// In order to export a given topic through multiple capabilities,
// Factory tracks existing topics internally.  See 'Join' for more details.
type Factory struct {
	cq  chan struct{}
	log log.Logger

	ps TopicJoiner

	mu sync.RWMutex
	wg sync.WaitGroup // blocks shutdown until all tasks are released
	ts map[string]*refCountedTopic
}

func New(ps TopicJoiner, opt ...Option) *Factory {
	var f = &Factory{
		cq: make(chan struct{}),
		ps: ps,
		ts: make(map[string]*refCountedTopic),
	}

	for _, option := range withDefault(opt) {
		option(f)
	}

	return f
}

func (f *Factory) Close() (err error) {
	if f != nil {
		select {
		case <-f.cq:
			err = fmt.Errorf("already %w", ErrClosed)
		default:
			close(f.cq)
			f.wg.Wait()
		}
	}

	return
}

func (f *Factory) New(p *server.Policy) PubSub {
	if p == nil {
		p = &defaultPolicy
	}

	return PubSub(api.PubSub_ServerToClient(f, p))
}

func (f *Factory) Join(ctx context.Context, call api.PubSub_join) error {
	call.Ack()

	name, err := call.Args().Name()
	if err != nil {
		return err
	}

	t, err := f.getOrCreate(name)
	if err != nil {
		return err
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetTopic(api.Topic_ServerToClient(t, &defaultPolicy))
}

func (f *Factory) getOrCreate(topic string) (*refCountedTopic, error) {
	f.mu.RLock()

	// fast path - already exists?
	if t, ok := f.ts[topic]; ok {
		defer f.mu.RUnlock()
		return t.AddRef(), nil
	}

	// slow path
	f.mu.RUnlock()
	f.mu.Lock()
	defer f.mu.Unlock()

	// topic may have been added while acquiring write-lock
	if t, ok := f.ts[topic]; ok {
		return t.AddRef(), nil
	}

	// join topic
	return f.joinTopic(topic)
}

// joinTopic and assign a refcounted topic to tm.ts.  Callers MUST hold a
// write-lock on f.mu.
func (f *Factory) joinTopic(topic string) (*refCountedTopic, error) {
	t, err := f.ps.Join(topic)
	if err != nil {
		return nil, err
	}

	f.wg.Add(1)
	release := func() {
		defer f.wg.Done()

		f.mu.Lock()
		defer f.mu.Unlock()

		delete(f.ts, topic)

		if err := t.Close(); err != nil {
			f.log.
				WithError(err).
				Errorf("unable to close topic %s", topic)
		}
	}

	rt := &refCountedTopic{
		log:     f.log.WithField("topic", topic),
		ctx:     ctxutil.C(f.cq),
		topic:   t,
		ref:     1,
		release: release,
	}

	f.ts[topic] = rt
	return rt, nil
}

type refCountedTopic struct {
	ctx   context.Context // root context for subscriptions
	log   log.Logger
	topic *pubsub.Topic

	mu  sync.Mutex
	ref int // number of refs from capnp.Client instances

	release capnp.ReleaseFunc // caller MUST hold mu
}

// AddRef MUST be called each time a new capnp.Client is
// created for t.
func (t *refCountedTopic) AddRef() *refCountedTopic {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.ref++
	return t
}

func (t *refCountedTopic) Release() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.ref--; t.ref == 0 {
		t.release()
	}
}

// The refCountedTopic is unique for each *pubsub.Topic, and is
// therefore shared across multiple capnp.Client instances. For
// this reason, Shutdown MAY be called multiple times.
func (t *refCountedTopic) Shutdown() { t.Release() }

func (t *refCountedTopic) Publish(ctx context.Context, call api.Topic_publish) error {
	if t.ctx.Err() != nil {
		return ErrClosed
	}

	b, err := call.Args().Msg()
	if err == nil {
		err = t.topic.Publish(ctx, b)
	}

	return err
}

func (t *refCountedTopic) Subscribe(_ context.Context, call api.Topic_subscribe) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	sub, err := t.topic.Subscribe()
	if err == nil {
		if t.ref == 0 {
			err = ErrClosed
		} else {
			t.ref++
			t.handle(call.Args(), sub)
		}
	}

	return err
}

func (t *refCountedTopic) handle(args api.Topic_subscribe_Params, sub *pubsub.Subscription) {
	h := subHandler{
		handler: args.Handler().AddRef(),
		buffer:  semaphore.NewWeighted(int64(args.BufSize())),
	}

	go func() {
		defer t.Release()
		defer sub.Cancel()
		defer h.handler.Release()

		h.Handle(t.ctx, sub)
	}()
}
