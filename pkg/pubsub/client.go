package pubsub

import (
	"context"

	capnp "capnproto.org/go/capnp/v3"

	casm "github.com/wetware/casm/pkg"
	chan_api "github.com/wetware/ww/internal/api/channel"
	api "github.com/wetware/ww/internal/api/pubsub"
)

// Joiner is a client capability that confers the right to join pubsub
// topics.  It is the dual to Router.
type Joiner api.Router

func (ps Joiner) Join(ctx context.Context, topic string) (Topic, capnp.ReleaseFunc) {
	f, release := (api.Router)(ps).Join(ctx, func(ps api.Router_join_Params) error {
		return ps.SetName(topic)
	})

	return Topic(f.Topic()), release
}

func (ps Joiner) AddRef() Joiner {
	return Joiner(capnp.Client(ps).AddRef())
}

func (ps Joiner) Release() {
	capnp.Client(ps).Release()
}

type FutureTopic api.Router_join_Results_Future

func (ft FutureTopic) Topic() Topic {
	return Topic(api.Router_join_Results_Future(ft).Topic())
}

func (ft FutureTopic) Struct() (Topic, error) {
	res, err := (api.Router_join_Results_Future)(ft).Struct()
	if err != nil {
		return Topic{}, err
	}

	return Topic(res.Topic()), nil
}

type Topic api.Topic

func (t Topic) AddRef() Topic {
	return Topic(api.Topic(t).AddRef())
}

func (t Topic) Release() {
	capnp.Client(t).Release()
}

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

func (t Topic) Subscribe(ctx context.Context) (Subscription, capnp.ReleaseFunc) {
	ctx, cancel := context.WithCancel(ctx)

	ch := make(handler, 256)
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

type Subscription casm.Iterator[[]byte]

func (sub Subscription) Next() []byte {
	b, _ := sub.Seq.Next()
	return b
}

func (sub Subscription) Err() error {
	return casm.Iterator[[]byte](sub).Err()
}

type handler chan []byte

func (ch handler) Params(ps api.Topic_subscribe_Params) error {
	ps.SetBuf(uint16(cap(ch)))
	return ps.SetChan(chan_api.Sender_ServerToClient(ch))
}

func (ch handler) Shutdown() { close(ch) }

func (ch handler) Next() (b []byte, ok bool) {
	b, ok = <-ch
	return
}

func (ch handler) Send(ctx context.Context, call chan_api.Sender_send) error {
	ptr, err := call.Args().Value()
	if err == nil {
		// It's okay to block here, since there is only one writer.
		// Back-pressure will be handled by the BBR flow-limiter.
		select {
		case ch <- ptr.Data():
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return err
}
