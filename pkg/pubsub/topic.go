package pubsub

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/flowcontrol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/util/stream"
	api "github.com/wetware/ww/internal/api/pubsub"
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

// NewStream provides an interface for publishing large volumes of data
// through a flow-controlled channel.   This will override the existing
// FlowLimiter.
func (t Topic) NewStream(ctx context.Context) Stream {
	// TODO:  use BBR once scheduler bug is fixed
	api.Topic(t).SetFlowLimiter(flowcontrol.NewFixedLimiter(1e6))

	cherr := make(chan error, 1)
	done := make(chan struct{})

	ctx, cancel := context.WithCancel(ctx)

	s := Stream{
		ctx:    ctx,
		cancel: cancel,
		cherr:  cherr,
		done:   done,
		topic:  t,
	}

	go func() {
		defer cancel()
		defer close(done)

		select {
		case s.err = <-cherr:
		case <-ctx.Done():
			s.err = ctx.Err()
		}
	}()

	return s
}

// PublishAsync submits a message for broadcast over the topic.  Unlike
// Publish, it returns a future.  This is useful when applications must
// publish a large volume of messages, and callers do not wish to spawn
// a goroutine for each call.  PublishAsync is nevertheless thread-safe.
func (t Topic) PublishAsync(ctx context.Context, b []byte) (casm.Future, capnp.ReleaseFunc) {
	f, release := api.Topic(t).Publish(ctx, message(b))
	return casm.Future(f), release
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

type Stream struct {
	ctx    context.Context
	cancel context.CancelFunc
	topic  Topic
	cherr  chan<- error
	done   <-chan struct{}
	err    error
}

func (s Stream) Publish(msg []byte) error {
	if err := s.ctx.Err(); err != nil {
		return err
	}

	f, release := s.topic.PublishAsync(s.ctx, msg)
	go func() {
		defer release()

		select {
		case <-f.Done():
			if err := f.Err(); err != nil {
				select {
				case s.cherr <- f.Err():
				default:
				}
			}

		case <-s.ctx.Done():
		}
	}()

	select {
	case <-s.done:
		return s.err
	default:
		return nil
	}
}

func (s Stream) Close() error {
	s.cancel()

	select {
	case <-s.done:
		return s.err
	default:
		return nil
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
	log   log.Logger
	topic *pubsub.Topic
	leave func(*pubsub.Topic) error
}

func (t topicServer) Shutdown() {
	if err := t.leave(t.topic); err != nil {
		panic(err) // invalid refcount
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
	handler := stream.New(consumer.Consume)
	for call.Go(); handler.Open(); t.log.Trace("message received") {
		handler.Call(ctx, bind(ctx, sub))
	}

	return nil
	// return handler.Wait()  // FIXME
}

func (t topicServer) subscribe(call MethodSubscribe) (*pubsub.Subscription, error) {
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
