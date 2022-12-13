package pubsub

import (
	"context"
	"sync"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/exp/clock"
	"capnproto.org/go/capnp/v3/flowcontrol/bbr"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/util/stream"
	chan_api "github.com/wetware/ww/internal/api/channel"
	api "github.com/wetware/ww/internal/api/pubsub"
	"github.com/wetware/ww/pkg/csp"
)

// Topic is the handle for a pubsub topic.  It is used to publish to
// the topic, and to manage subscriptions.
type Topic api.Topic

func (t Topic) AddRef() Topic {
	return Topic(api.Topic(t).AddRef())
}

func (t Topic) Release() {
	capnp.Client(t).Release()
}

// Name returns the name of the pubsub topic.  This is guaranteed never
// to change, so callers MAY cache results locally.
func (t Topic) Name(ctx context.Context) (string, error) {
	f, release := api.Topic(t).Name(ctx, nil)
	defer release()

	res, err := f.Struct()
	if err != nil {
		return "", err
	}

	return res.Name()
}

// Publish a message synchronously.  This is a convenience function that
// is equivalent to calling PublishAsync() and blocking on the future it
// returns. The drawback is that each call will block until it completes
// a round-trip.  It is safe to call Publish concurrently.
func (t Topic) Publish(ctx context.Context, b []byte) error {
	f, release := t.PublishAsync(ctx, b)
	defer release()

	return f.Err()
}

// PublishAsync submits a message for broadcast over the topic.  Unlike
// Publish, it returns a future.  This is useful when applications must
// publish a large volume of messages, and callers do not wish to spawn
// a goroutine for each call.  PublishAsync is nevertheless thread-safe.
func (t Topic) PublishAsync(ctx context.Context, b []byte) (casm.Future, capnp.ReleaseFunc) {
	f, release := api.Topic(t).Publish(ctx, message(b))
	return casm.Future(f), release
}

func message(b []byte) func(api.Topic_publish_Params) error {
	return func(ps api.Topic_publish_Params) error {
		return ps.SetMsg(b)
	}
}

// Subscribe to the topic.  Callers MUST call the provided ReleaseFunc
// when finished with the subscription, or a resource leak will occur.
func (t Topic) Subscribe(ctx context.Context) (Subscription, capnp.ReleaseFunc) {
	ctx, cancel := context.WithCancel(ctx)

	ch := make(handler, 16)
	f, release := api.Topic(t).Subscribe(ctx, ch.Params)

	sub := Subscription{
		Future: casm.Future(f),
		Seq:    ch,
	}

	if !api.Topic(t).IsValid() {
		close(ch) // necessary in case capnp.Client is wrong in the client side (e.g. is nil)
	}

	return sub, func() {
		cancel()
		release()
	}
}

/*
	Topic Manager
*/

// topicManager is responsible for refcounting *pubsub.Topic instances.
type topicManager struct {
	mu     sync.Mutex
	topics map[string]api.Topic
}

func (tm *topicManager) GetOrCreate(log log.Logger, ps TopicJoiner, name string) (api.Topic, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// do we have one, already?
	if t := tm.topics[name]; (t != api.Topic{}) {
		return t.AddRef(), nil
	}

	// slow path...

	return tm.join(log, ps, name)
}

// join a topic and add it to the map.  Caller MUST hold mu.
func (tm *topicManager) join(log log.Logger, ps TopicJoiner, name string) (topic api.Topic, err error) {
	var t *pubsub.Topic
	if t, err = ps.Join(name); err == nil {
		topic = tm.asCapability(log, t)
	}

	return

}

// returns a capability for the supplied topic.  Caller MUST hold mu.
func (tm *topicManager) asCapability(log log.Logger, t *pubsub.Topic) api.Topic {
	if tm.topics == nil {
		tm.topics = make(map[string]api.Topic)
	}

	server := tm.newTopicServer(log, t)
	topic := api.Topic_ServerToClient(server)

	tm.topics[t.String()] = topic

	return topic
}

// closes the topic and removes it from the map. Caller MUST hold mu.
func (tm *topicManager) leave(t *pubsub.Topic) error {
	// NOTE: we MUST hold mu until leave() returns. If we release before
	// t has been closed, a concurrent call might try to join t, causing
	// a "topic already exists" error.
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// The defer isn't actually be necessary, but it ensures stale topics
	// aren't left in the map if t.Close() were to panic.
	defer delete(tm.topics, t.String())

	// MUST happen before tm.mu.Unlock()
	return t.Close()
}

/*
	Topic Server
*/

type topicServer struct {
	log   log.Logger
	topic *pubsub.Topic
	leave func(*pubsub.Topic) error
}

func (tm *topicManager) newTopicServer(log log.Logger, t *pubsub.Topic) *topicServer {
	return &topicServer{
		log:   log.WithField("topic", t),
		topic: t,
		leave: tm.leave,
	}
}

func (t topicServer) Shutdown() {
	if err := t.leave(t.topic); err != nil {
		t.log.Fatal(err) // refcounting error; application state is invalid.
	}
}

func (t topicServer) Name(_ context.Context, call MethodName) error {
	res, err := call.AllocResults()
	if err == nil {
		err = res.SetName(t.topic.String())
	}
	return err
}

func (t topicServer) Publish(ctx context.Context, call MethodPublish) error {
	b, err := call.Args().Msg()
	if err != nil {
		return err
	}

	// Copy the message data.  t.topic.Publish is asynchronous, and the
	// segment will be zeroed when Send returns.
	msg := make([]byte, len(b))
	copy(msg, b)

	// The call to t.topic.Publish() may block if the underlying router
	// is not in the 'ready' state (e.g. discovering peers).  It's okay
	// to block the RPC handler in such cases.  BBR will detect this as
	// latency and automatically throttle the number of in-flight calls.
	// Better to avoid spawning a goroutine each time we publish.
	return t.topic.Publish(ctx, msg)
}

func (t topicServer) Subscribe(ctx context.Context, call MethodSubscribe) error {
	sub, err := t.subscribe(ctx, call.Args())
	if err != nil {
		return err
	}
	defer sub.Cancel()

	sender := call.Args().Chan()
	sender.SetFlowLimiter(bbr.NewLimiter(clock.System))

	handler := stream.New(sender.Send)
	next := bind(ctx, sub)

	t.log.Trace("registered subscription handler")
	defer t.log.Trace("unregistered subscription handler")

	// forward messages to the callback channel
	for call.Go(); handler.Open(); t.log.Trace("message received") {
		handler.Call(ctx, next)
	}

	return handler.Wait()
}

func (t topicServer) subscribe(ctx context.Context, args api.Topic_subscribe_Params) (sub *pubsub.Subscription, err error) {
	if err = ctx.Err(); err == nil {
		bufsize := int(args.Buf())
		sub, err = t.topic.Subscribe(pubsub.WithBufferSize(bufsize))
	}

	return
}

func bind(ctx context.Context, sub *pubsub.Subscription) csp.Value {
	return func(ps chan_api.Sender_send_Params) error {
		msg, err := sub.Next(ctx)
		if err != nil {
			return err
		}

		return capnp.Struct(ps).SetData(0, msg.Data)
	}
}
