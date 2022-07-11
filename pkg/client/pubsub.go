package client

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"github.com/wetware/ww/pkg/vat/cap/pubsub"
)

type Topic struct {
	name   string
	Client *capnp.Client
}

// NewTopic populates a Topic with the supplied name and capability.
// It does not validate the name.
func NewTopic(c *capnp.Client, name string) Topic {
	return Topic{
		name:   name,
		Client: c,
	}
}

// ResolveTopic populates a Topic from a raw capability client. It performs
// an RPC call to determine the topic name and populates t with the result.
func ResolveTopic(ctx context.Context, c *capnp.Client) (t Topic, err error) {
	t.Client = c
	t.name, err = pubsub.Topic{Client: c}.Name(ctx)
	return
}

func (t Topic) String() string { return t.name }

func (t Topic) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"topic": t.name,
	}
}

func (t Topic) AddRef() Topic {
	return Topic{
		name:   t.name,
		Client: t.Client.AddRef(),
	}
}

func (t Topic) Release() { t.Client.Release() }

func (t Topic) Publish(ctx context.Context, msg []byte) error {
	return pubsub.Topic{Client: t.Client}.Publish(ctx, msg)
}

func (t Topic) Subscribe(ctx context.Context) (Subscription, error) {
	out := make(chan []byte, 32)

	release, err := pubsub.Topic{Client: t.Client}.Subscribe(ctx, out)
	return Subscription{
		name:   t.name,
		cancel: release,
		c:      out,
	}, err
}

type Subscription struct {
	name   string
	cancel func()
	c      <-chan []byte
}

func (s Subscription) String() string { return s.name }

func (s Subscription) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"topic": s.name,
	}
}

func (s Subscription) Cancel() { s.cancel() }

func (s Subscription) Next(ctx context.Context) ([]byte, error) {
	select {
	case b, ok := <-s.c:
		if ok {
			return b, nil
		}

		return nil, ErrDisconnected

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
