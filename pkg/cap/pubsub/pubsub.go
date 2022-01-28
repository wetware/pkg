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
	syncutil "github.com/lthibault/util/sync"
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
	TopicJoiner
	Log log.Logger

	mu sync.RWMutex
	ts map[string]*topicRecord
}

func (f *Factory) Close() (err error) {
	// capability provided?
	if f != nil {
		f.mu.Lock()
		defer f.mu.Unlock()

		for _, topic := range f.ts {
			err = multierr.Append(err, topic.Close())
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

	return res.SetTopic(api.Topic_ServerToClient(
		f.newTopicCap(t),
		&defaultPolicy))
}

func (f *Factory) getOrCreate(topic string) (t *pubsub.Topic, err error) {
	f.mu.RLock()

	// fast path - already exists?
	var ok bool
	if t, ok = f.getAndIncr(topic); ok {
		defer f.mu.RUnlock()
		return
	}

	// slow path
	f.mu.RUnlock()
	f.mu.Lock()
	defer f.mu.Unlock()

	// topic may have been added while acquiring write-lock
	if t, ok = f.getAndIncr(topic); !ok {
		// initialize map?
		if f.ts == nil {
			f.ts = make(map[string]*topicRecord, 1)
		}

		t, err = f.joinTopic(topic)
	}

	return
}

// getAndIncr returns the designated topic and increments its refcount,
// if it exists.  Callers MUST hold f.mu.
func (f *Factory) getAndIncr(topic string) (t *pubsub.Topic, ok bool) {
	var rec *topicRecord
	if rec, ok = f.ts[topic]; ok {
		rec.Ref.Incr()
		t = rec.Topic
	}

	return
}

// joinTopic and assign a refcounted topic to tm.ts.  Callers MUST hold a
// write-lock on f.mu.
func (f *Factory) joinTopic(topic string) (t *pubsub.Topic, err error) {
	if t, err = f.TopicJoiner.Join(topic); err == nil {
		f.ts[topic] = &topicRecord{
			Ref:   1,
			Topic: t,
		}
	}

	return
}

// Caller MUST hold a write-lock on f.mu.
func (f *Factory) leaveTopic(topic string) error {
	if rec, ok := f.ts[topic]; ok {
		if rec.Ref.Decr() == 0 {
			delete(f.ts, topic)
			return rec.Topic.Close() // don't decorate error (see rec.Close)
		}

		return nil
	}

	return errors.New("not found")
}

type topicRecord struct {
	Ref syncutil.Ctr
	*pubsub.Topic
}

func (rec topicRecord) Close() (err error) {
	if err = rec.Topic.Close(); err != nil {
		err = fmt.Errorf("close %s: %w", rec, err)
	}

	return
}

type topic struct {
	*pubsub.Topic
	cq      ctxutil.C
	release capnp.ReleaseFunc
}

func (f *Factory) newTopicCap(t *pubsub.Topic) topic {
	var cq = make(chan struct{})

	release := func() {
		close(cq)

		f.mu.Lock()
		defer f.mu.Unlock()

		if err := f.leaveTopic(t.String()); err != nil {
			if f.Log == nil {
				f.Log = log.New(log.WithLevel(log.ErrorLevel))
			}

			f.Log.
				WithError(err).
				Errorf("failed to release topic '%s'", t)
		}
	}

	return topic{
		Topic:   t,
		cq:      cq,
		release: release,
	}
}

func (t topic) Shutdown() {
	t.release()
}

func (t topic) Publish(ctx context.Context, call api.Topic_publish) error {
	b, err := call.Args().Msg()
	if err == nil {
		err = t.Topic.Publish(ctx, b)
	}

	return err
}

func (t topic) Subscribe(_ context.Context, call api.Topic_subscribe) error {
	sub, err := t.Topic.Subscribe()
	if err == nil {
		go subHandler{
			handler: call.Args().Handler().AddRef(),
			buffer:  semaphore.NewWeighted(int64(call.Args().BufSize())),
		}.Handle(t.cq, sub)
	}

	return err
}
