package pubsub

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"

	"capnproto.org/go/capnp/v3/server"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	api "github.com/wetware/ww/internal/api/pubsub"
	"go.uber.org/multierr"
	"golang.org/x/sync/semaphore"
)

var ErrClosed = errors.New("closed")

type TopicJoiner interface {
	Join(string, ...pubsub.TopicOpt) (*pubsub.Topic, error)
}

var defaultPolicy = server.Policy{
	// HACK:  raise MaxConcurrentCalls to mitigate known deadlock condition.
	//        https://github.com/capnproto/go-capnproto2/issues/189
	MaxConcurrentCalls: 64,
	AnswerQueueSize:    64,
}

// Factory wraps a PubSub and provides a NewCap() factory method that
// returns a client capability for the pubsub.
//
// In order to export a given topic through multiple capabilities,
// Factory tracks existing topics internally.  See 'Join' for more details.
type Factory struct {
	cq      chan struct{}
	ts      topicManager
	onJoin  chan evtTopicJoinRequested
	onLeave chan evtTopicReleased
}

func New(ps TopicJoiner) Factory {
	f := Factory{
		ts:      newTopicManager(ps),
		cq:      make(chan struct{}),
		onJoin:  make(chan evtTopicJoinRequested),
		onLeave: make(chan evtTopicReleased), // TODO:  buffer? (finalizer is single-threaded)
	}

	go f.run()

	return f
}

func (f Factory) run() {
	for {
		select {
		case evt := <-f.onJoin:
			if t, err := f.ts.GetOrCreate(evt.Topic); err != nil {
				evt.Err <- err
			} else {
				evt.Res <- t
			}

		case evt := <-f.onLeave:
			evt.Err <- f.ts.Release(evt.Topic)

		case <-f.cq:
			return
		}
	}
}

func (f Factory) Close() error {
	// capability not provided?
	if f.cq == nil {
		return nil
	}

	select {
	case <-f.cq:
		return fmt.Errorf("already %w", ErrClosed) // support errors.Is
	default:
		return f.ts.Close()
	}
}

func (f Factory) New(p *server.Policy) PubSub {
	if p == nil {
		p = &defaultPolicy
	}

	return PubSub(api.PubSub_ServerToClient(f, p))
}

func (f Factory) Join(ctx context.Context, call api.PubSub_join) error {
	call.Ack()

	name, err := call.Args().Name()
	if err != nil {
		return err
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	topic := make(chan *pubsub.Topic, 1) // TODO:  pool
	cherr := make(chan error, 1)

	select {
	case f.onJoin <- evtTopicJoinRequested{
		Topic: name,
		Res:   topic,
		Err:   cherr,
	}:
	case <-f.cq:
		return ErrClosed
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case t := <-topic:
		return res.SetTopic(api.Topic_ServerToClient(
			f.newTopicCap(t),
			&defaultPolicy))
	case err := <-cherr:
		return err
	case <-f.cq:
		return ErrClosed
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (f Factory) newTopicCap(t *pubsub.Topic) topicCap {
	var r = &topicReleaser{
		cq:     f.cq,
		signal: f.onLeave,
	}

	// Ensure we decrement the refcount, even if the user forgets to
	// release the topic.
	runtime.SetFinalizer(r, func(r releaser) {
		_ = r.Release(t)
	})

	return topicCap{
		t:   t,
		ref: r, // r MUST be a pointer type (see SetFinalizer)
	}
}

type refCountedTopic struct {
	Ref uint16
	T   *pubsub.Topic
}

type topicManager struct {
	ps TopicJoiner
	ts map[string]*refCountedTopic
}

func newTopicManager(ps TopicJoiner) topicManager {
	return topicManager{
		ps: ps,
		ts: make(map[string]*refCountedTopic),
	}
}

func (tm topicManager) Close() (err error) {
	for _, topic := range tm.ts {
		err = multierr.Append(err, topic.T.Close())
	}

	return
}

func (tm topicManager) GetOrCreate(topic string) (*pubsub.Topic, error) {
	// fast path - already exists?
	if rt, ok := tm.ts[topic]; ok {
		rt.Ref++
		return rt.T, nil
	}

	// slow path - join topic
	t, err := tm.ps.Join(topic)
	if err == nil {
		tm.ts[topic] = &refCountedTopic{
			Ref: 1,
			T:   t,
		}
	}

	return t, err
}

func (tm topicManager) Release(topic string) error {
	if rt, ok := tm.ts[topic]; ok {
		if rt.Ref--; rt.Ref == 0 {
			defer delete(tm.ts, topic)
			return rt.T.Close()
		}

		return nil
	}

	return fmt.Errorf("refcount error: topic '%s' not in manager", topic)
}

type evtTopicJoinRequested struct {
	Topic string
	Res   chan<- *pubsub.Topic
	Err   chan<- error
}

type evtTopicReleased struct {
	Topic string
	Err   chan<- error
}

type topicCap struct {
	t   *pubsub.Topic
	ref releaser
}

func (tc topicCap) Publish(ctx context.Context, call api.Topic_publish) error {
	call.Ack()

	b, err := call.Args().Msg()
	if err != nil {
		return err
	}

	return tc.t.Publish(ctx, b)
}

func (tc topicCap) Subscribe(ctx context.Context, call api.Topic_subscribe) error {
	call.Ack()

	sub, err := tc.t.Subscribe()
	if err == nil {
		go subHandler{
			handler: call.Args().Handler().AddRef(),
			buffer:  semaphore.NewWeighted(int64(call.Args().BufSize())),
		}.
			Handle(context.TODO(), sub)
	}

	return err
}

// We use this interface to enforce the use of a pointer type in topicCap.release.
// This is required in order for runtime.SetFinalizer to function correctly.
type releaser interface {
	Release(*pubsub.Topic) error
}

type topicReleaser struct {
	once   sync.Once
	cq     <-chan struct{}
	signal chan<- evtTopicReleased
}

// NOTE:  pointer method is ESSENTIAL for enforcing SetFinalizer constraint.
//        It forces the use of a *topicReleaser in topicCap.release.
func (r *topicReleaser) Release(t *pubsub.Topic) (err error) {
	r.once.Do(func() {
		cherr := make(chan error, 1) // TODO: pool?

		evt := evtTopicReleased{
			Topic: t.String(),
			Err:   cherr,
		}

		select {
		case r.signal <- evt:
			select {
			case err = <-cherr:
			case <-r.cq: // closing; swallow error
			}

		case <-r.cq:
		}
	})

	return
}
