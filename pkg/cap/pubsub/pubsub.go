package pubsub

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"

	"capnproto.org/go/capnp/v3/server"
	"github.com/jbenet/goprocess"
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
	ps      TopicJoiner
	cq      chan struct{}
	onJoin  chan evtTopicJoinRequested
	onLeave chan evtTopicReleased
}

func New(ps TopicJoiner) Factory {
	return Factory{
		ps:      ps,
		cq:      make(chan struct{}),
		onJoin:  make(chan evtTopicJoinRequested),
		onLeave: make(chan evtTopicReleased), // TODO:  buffer? (finalizer is single-threaded)
	}
}

func (s Factory) Run() goprocess.ProcessFunc {
	return func(proc goprocess.Process) {
		defer close(s.cq)

		var ts = make(topicManager)
		proc.SetTeardown(ts.Close)

		for {
			select {
			case evt := <-s.onJoin:
				if t, err := ts.GetOrCreate(s.ps, evt.Topic); err != nil {
					evt.Err <- err
				} else {
					evt.Res <- t
				}

			case evt := <-s.onLeave:
				evt.Err <- ts.Release(evt.Topic)

			case <-proc.Closing():
				return
			}
		}
	}
}

func (s Factory) New(p *server.Policy) PubSub {
	if p == nil {
		p = &defaultPolicy
	}

	return PubSub(api.PubSub_ServerToClient(s, p))
}

func (s Factory) Join(ctx context.Context, call api.PubSub_join) error {
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
	case s.onJoin <- evtTopicJoinRequested{
		Topic: name,
		Res:   topic,
		Err:   cherr,
	}:
	case <-s.cq:
		return ErrClosed
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case t := <-topic:
		return res.SetTopic(api.Topic_ServerToClient(
			s.newTopicCap(t),
			&defaultPolicy))
	case err := <-cherr:
		return err
	case <-s.cq:
		return ErrClosed
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s Factory) newTopicCap(t *pubsub.Topic) topicCap {
	var r = &topicReleaser{
		cq:     s.cq,
		signal: s.onLeave,
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

type topicManager map[string]*refCountedTopic

func (ts topicManager) Close() (err error) {
	for _, topic := range ts {
		err = multierr.Append(err, topic.T.Close())
	}

	return
}

func (ts topicManager) GetOrCreate(ps TopicJoiner, topic string) (*pubsub.Topic, error) {
	// fast path - already exists?
	if rt, ok := ts[topic]; ok {
		rt.Ref++
		return rt.T, nil
	}

	// slow path - join topic
	t, err := ps.Join(topic)
	if err == nil {
		ts[topic] = &refCountedTopic{
			Ref: 1,
			T:   t,
		}
	}

	return t, err
}

func (ts topicManager) Release(topic string) error {
	if rt, ok := ts[topic]; ok {
		if rt.Ref--; rt.Ref == 0 {
			defer delete(ts, topic)
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
