package client

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"github.com/wetware/ww/pkg/cap/pubsub"
)

type Topic struct {
	name    string
	f       pubsub.FutureTopic
	Release capnp.ReleaseFunc
}

func (t Topic) String() string { return t.name }

func (t Topic) Publish(ctx context.Context, msg []byte) error {
	return t.f.Topic().Publish(ctx, msg)
}

func (t Topic) Subscribe(ctx context.Context) (Subscription, error) {
	topic, err := t.f.Struct()
	if err != nil {
		return Subscription{}, err
	}

	out := make(chan []byte, 32)

	cancel, err := topic.Subscribe(ctx, out)
	return Subscription{
		C:      out,
		Cancel: cancel,
	}, err

}

type Subscription struct {
	C      <-chan []byte
	Cancel func()
}
