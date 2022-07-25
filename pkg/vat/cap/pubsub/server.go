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
	"github.com/wetware/ww/pkg/vat/cap/channel"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

var ErrClosed = errors.New("closed")

var defaultPolicy = server.Policy{
	MaxConcurrentCalls: 64,
}

type TopicJoiner interface {
	Join(string, ...pubsub.TopicOpt) (*pubsub.Topic, error)
}

// Provider wraps a PubSub and provides vat.ClientProvider.
//
// In order to export a given topic through multiple capabilities,
// Provider tracks existing topics internally.  See 'Join' for more details.
type Provider struct {
	cq  chan struct{}
	log log.Logger

	ps TopicJoiner

	mu sync.RWMutex
	wg sync.WaitGroup // blocks shutdown until all tasks are released
	ts map[string]*refCountedTopic
}

func New(ns string, ps TopicJoiner, opt ...Option) *Provider {
	var f = &Provider{
		cq: make(chan struct{}),
		ps: ps,
		ts: make(map[string]*refCountedTopic),
	}

	for _, option := range withDefault(opt) {
		option(f)
	}

	return f
}

func (p *Provider) Close() (err error) {
	if p != nil {
		select {
		case <-p.cq:
			err = fmt.Errorf("already %w", ErrClosed)
		default:
			close(p.cq)
			p.wg.Wait()
		}
	}

	return
}

func (p *Provider) Client() capnp.Client {
	return api.PubSub_ServerToClient(p, &defaultPolicy).Client
}

func (p *Provider) Join(ctx context.Context, call api.PubSub_join) error {
	call.Ack()

	name, err := call.Args().Name()
	if err != nil {
		return err
	}

	t, err := p.getOrCreate(name)
	if err != nil {
		return err
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetTopic(api.Topic_ServerToClient(t, &defaultPolicy))
}

func (p *Provider) getOrCreate(topic string) (*refCountedTopic, error) {
	p.mu.RLock()

	// fast path - already exists?
	if t, ok := p.ts[topic]; ok {
		defer p.mu.RUnlock()
		return t.AddRef(), nil
	}

	// slow path
	p.mu.RUnlock()
	p.mu.Lock()
	defer p.mu.Unlock()

	// topic may have been added while acquiring write-lock
	if t, ok := p.ts[topic]; ok {
		return t.AddRef(), nil
	}

	// join topic
	return p.joinTopic(topic)
}

// joinTopic and assign a refcounted topic to tm.ts.  Callers MUST hold a
// write-lock on f.mu.
func (p *Provider) joinTopic(topic string) (*refCountedTopic, error) {
	t, err := p.ps.Join(topic)
	if err != nil {
		return nil, err
	}

	p.wg.Add(1)
	release := func() {
		defer p.wg.Done()

		p.mu.Lock()
		defer p.mu.Unlock()

		delete(p.ts, topic)

		if err := t.Close(); err != nil {
			p.log.
				WithError(err).
				Errorf("unable to close topic %s", topic)
		}
	}

	rt := &refCountedTopic{
		log:     p.log.WithField("topic", topic),
		ctx:     ctxutil.C(p.cq),
		topic:   t,
		ref:     1,
		release: release,
	}

	p.ts[topic] = rt
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

func (t *refCountedTopic) Name(ctx context.Context, call api.Topic_name) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetName(t.topic.String())
}

func (t *refCountedTopic) Publish(ctx context.Context, call api.Topic_publish) error {
	if t.ctx.Err() != nil {
		return ErrClosed
	}

	b, err := call.Args().Msg()
	if err == nil {
		call.Ack()
		err = t.topic.Publish(ctx, b)
	}

	return err
}

func (t *refCountedTopic) Subscribe(_ context.Context, call api.Topic_subscribe) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.ref == 0 {
		return ErrClosed
	}

	s, err := t.subscribe(call.Args())
	if err == nil {
		go s.Stream()
	}

	return err
}

// subscribe creates a flow-controlled subscription, capable of streaming
// messages through args.Chan().
//
// Increments t.ref.  Callers MUST hold t.mu.
func (t *refCountedTopic) subscribe(args api.Topic_subscribe_Params) (s subscription, err error) {
	subOpts, err := args.Opts()
	if err != nil {
		return s, err
	}

	if s.sub, err = t.topic.Subscribe(pubsub.WithBufferSize(int(subOpts.BufferSize()))); err == nil {
		s.ch = channel.Sender(args.Chan().AddRef())
		s.ch.Client.SetFlowLimiter(newFlowLimiter(subOpts.BufferSize()))

		t.ref++
		s.t = t
	}

	return
}

type subscription struct {
	sub *pubsub.Subscription
	ch  channel.Sender
	t   *refCountedTopic
}

func (s *subscription) release() {
	s.sub.Cancel()
	s.ch.Release()
	s.t.Release()
}

func (s *subscription) Stream() {
	defer s.release()

	ctx, cancel := context.WithCancel(s.t.ctx)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	for ctx.Err() == nil {
		m, err := s.sub.Next(ctx)
		if err != nil {
			return
		}

		g.Go(s.send(ctx, m))
	}
}

func (s *subscription) send(ctx context.Context, m *pubsub.Message) func() error {
	f, release := s.ch.Send(ctx, channel.Data(m.Data))
	return func() error {
		defer release()
		return f.Err()
	}
}

type flowLimiter semaphore.Weighted

func newFlowLimiter(limit int64) *flowLimiter {
	return (*flowLimiter)(semaphore.NewWeighted(limit))
}

func (f *flowLimiter) StartMessage(ctx context.Context, size uint64) (gotResponse func(), err error) {
	if err = (*semaphore.Weighted)(f).Acquire(ctx, 1); err == nil {
		gotResponse = func() { (*semaphore.Weighted)(f).Release(1) }
	}

	return
}
