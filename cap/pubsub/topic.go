//go:generate mockgen -source=topic.go -destination=test/topic.go -package=test_pubsub

package pubsub

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/flowcontrol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	api "github.com/wetware/pkg/api/pubsub"
	"github.com/wetware/pkg/util/casm"
)

// Logger is used for logging by the RPC system. Each method logs
// messages at a different level, but otherwise has the same semantics:
//
//   - Message is a human-readable description of the log event.
//   - Args is a sequenece of key, value pairs, where the keys must be strings
//     and the values may be any type.
//   - The methods may not block for long periods of time.
//
// This interface is designed such that it is satisfied by *slog.Logger.
type Logger interface {
	Debug(message string, args ...any)
	Info(message string, args ...any)
	Warn(message string, args ...any)
	Error(message string, args ...any)
}

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

// Publish a message asynchronously.  The first error encountered will
// be returned by all subsequent calls to Publish().
func (t Topic) Publish(ctx context.Context, b []byte) error {
	return api.Topic(t).Publish(ctx, message(b))
}

// Subscribe to the topic.  Callers MUST call the provided ReleaseFunc
// when finished with the subscription, or a resource leak will occur.
func (t Topic) Subscribe(ctx context.Context) (Subscription, capnp.ReleaseFunc) {
	// Aborting early simplifies the lifecycle logic for the handler.
	// We still invoke api.Topic.Subscribe() in order to report the
	// null capability error to the caller.
	if !api.Topic(t).IsValid() {
		f, release := api.Topic(t).Subscribe(ctx, nil)
		return Subscription{Future: casm.Future(f)}, release
	}

	// The user needs to be able to abort the call, so we derive a
	// context and wrap its CancelFunc in the release function.
	ctx, cancel := context.WithCancel(ctx)

	var (
		c          = make(consumer, 16)
		f, release = api.Topic(t).Subscribe(ctx, c.Params)
	)

	return Subscription{
			Future: casm.Future(f),
			Seq:    c,
		}, func() {
			cancel()
			release()
		}
}

func message(b []byte) func(api.Topic_publish_Params) error {
	return func(ps api.Topic_publish_Params) error {
		return ps.SetMsg(b)
	}
}

/*
	Topic Server
*/

type topicServer struct {
	log   Logger
	topic *pubsub.Topic
	leave func(*pubsub.Topic) error
}

func (t topicServer) Shutdown() {
	if err := t.leave(t.topic); err != nil {
		panic(err) // invalid refcount
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

func (t topicServer) Subscribe(ctx context.Context, call api.Topic_subscribe) error {
	// Subscribe can't be called with a released client, so there's no need to
	// check the context before subscribing to the libp2p topic. We will catch
	// context cancellations in the stream handler.

	sub, err := t.subscribe(call)
	if err != nil {
		return err
	}
	defer sub.Cancel()

	consumer := call.Args().Consumer()
	consumer.SetFlowLimiter(flowcontrol.NewFixedLimiter(1e6)) // TODO:  use BBR once scheduler bug is fixed

	t.log.Debug("registered subscription handler")
	defer t.log.Debug("unregistered subscription handler")

	// forward messages to the callback channel
	for call.Go(); ctx.Err() == nil; t.log.Debug("message received") {
		if err = consumer.Consume(ctx, bind(ctx, sub)); err != nil {
			break
		}
	}

	return consumer.WaitStreaming()
}

func (t topicServer) subscribe(call api.Topic_subscribe) (*pubsub.Subscription, error) {
	bufsize := int(call.Args().Buf())
	return t.topic.Subscribe(pubsub.WithBufferSize(bufsize))
}

// bind the libp2p subscription to the handler.  Note that
// we MUST NOT call sub.Next() inside of the callback, or
// capnp will deadlock.
func bind(ctx context.Context, sub *pubsub.Subscription) func(api.Topic_Consumer_consume_Params) error {
	msg, err := sub.Next(ctx)
	if err != nil {
		return failure(err)
	}

	return func(ps api.Topic_Consumer_consume_Params) error {
		return ps.SetMsg(msg.Data)
	}
}

func failure(err error) func(api.Topic_Consumer_consume_Params) error {
	return func(api.Topic_Consumer_consume_Params) error {
		return err
	}
}
