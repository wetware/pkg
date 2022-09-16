package pubsub

import (
	"context"
	"errors"
	"sync"

	capnp "capnproto.org/go/capnp/v3"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"

	"github.com/wetware/casm/pkg/util/stream"
	chan_api "github.com/wetware/ww/internal/api/channel"
	api "github.com/wetware/ww/internal/api/pubsub"
	"github.com/wetware/ww/pkg/channel"
)

var ErrClosed = errors.New("closed")

type TopicJoiner interface {
	Join(string, ...pubsub.TopicOpt) (*pubsub.Topic, error)
}

type Router struct {
	Log         log.Logger
	TopicJoiner TopicJoiner
	topics      *topicManager
}

func (r *Router) PubSub() Joiner {
	return Joiner(api.Router_ServerToClient(r))
}

func (r *Router) Client() capnp.Client {
	return capnp.Client(r.PubSub())
}

func (r *Router) Join(ctx context.Context, call api.Router_join) error {
	r.initialize() // not thread-safe

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	name, err := call.Args().Name()
	if err != nil {
		return err
	}

	t, err := r.topics.GetOrCreate(r.TopicJoiner, name)
	if err != nil {
		return err
	}

	logger := r.Log.WithField("topic", t.String())
	defer logger.Trace("acquired topic ref")

	return res.SetTopic(api.Topic_ServerToClient(topicServer{
		log:     logger,
		manager: r.topics,
		topic:   t,
	}))
}

func (r *Router) initialize() {
	if r.topics == nil {
		if r.Log == nil {
			r.Log = log.New()
		}

		r.topics = newTopicManager()
	}
}

type topicServer struct {
	log     log.Logger
	manager *topicManager
	topic   *pubsub.Topic
}

func (t topicServer) Shutdown() {
	if err := t.manager.Release(t.topic); err != nil {
		t.log.WithError(err).Error("failed to release topic ref")
	} else {
		t.log.Trace("released topic ref")
	}
}

func (t topicServer) Name(_ context.Context, call api.Topic_name) error {
	res, err := call.AllocResults()
	if err == nil {
		err = res.SetName(t.topic.String())
	}
	return err
}

func (t topicServer) Publish(ctx context.Context, call api.Topic_publish) error {
	b, err := call.Args().Msg()
	if err == nil {
		// The call to t.topic.Publish() may block if the underlying router
		// is not in the 'ready' state (e.g. discovering peers).  It's okay
		// to block the RPC handler in such cases.  BBR will detect this as
		// latency and automatically throttle the number of in-flight calls.
		// Better to avoid spawning a goroutine each time we publish.
		err = t.topic.Publish(ctx, b)
	}

	return err
}

func (t topicServer) Subscribe(ctx context.Context, call api.Topic_subscribe) error {
	sub, err := t.manager.Subscribe(t.topic, call.Args().Buf())
	if err != nil {
		return err
	}
	defer t.manager.Release(t.topic)
	defer sub.Cancel()

	sender := call.Args().Chan()
	handler := stream.New(sender.Send)

	t.log.Trace("registered subscription handler")
	defer t.log.Trace("unregistered subscription handler")

	// forward messages to the callback channel
	for call.Ack(); handler.Open(); t.log.Trace("message received") {
		handler.Call(ctx, bind(ctx, sub))
	}

	return handler.Wait()
}

func bind(ctx context.Context, sub *pubsub.Subscription) channel.Value {
	msg, err := sub.Next(ctx)
	if err != nil {
		return failure(err)
	}

	return channel.Data(msg.Data)
}

func failure(err error) channel.Value {
	return func(chan_api.Sender_send_Params) error {
		return err
	}
}

type topicManager struct {
	mu     sync.Mutex
	topics map[string]*pubsub.Topic
	refs   map[string]int
}

func newTopicManager() *topicManager {
	return &topicManager{
		topics: make(map[string]*pubsub.Topic),
		refs:   make(map[string]int),
	}
}

func (tm *topicManager) GetOrCreate(ps TopicJoiner, name string) (t *pubsub.Topic, err error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	var ok bool
	if t, ok = tm.topics[name]; !ok {
		if t, err = ps.Join(name); err == nil {
			tm.topics[name] = t // add to active topics
		}
	}

	if err == nil {
		tm.refs[name]++ // increment the ref count
	}

	return
}

func (tm *topicManager) Release(t *pubsub.Topic) (err error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	name := t.String()

	if tm.refs[name]--; tm.refs[name] == 0 {
		delete(tm.topics, name)
		delete(tm.refs, name)
		err = t.Close()
	}

	return
}

func (tm *topicManager) Subscribe(t *pubsub.Topic, buf uint16) (*pubsub.Subscription, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	s, err := t.Subscribe(pubsub.WithBufferSize(int(buf)))
	if err == nil {
		tm.refs[t.String()]++
	}

	return s, err
}
