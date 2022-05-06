package client

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"github.com/wetware/ww/pkg/cap/pubsub"
)

type Topic struct {
	Name   string
	Client *capnp.Client
	done   <-chan struct{} // rpc.Conn.Done()
}

func (t Topic) String() string { return t.Name }

func (t Topic) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"topic": t.Name,
	}
}

func (t Topic) AddRef() Topic {
	return Topic{
		Name:   t.Name,
		Client: t.Client.AddRef(),
		done:   t.done,
	}
}

func (t Topic) Release() { t.Client.Release() }

func (t Topic) Publish(ctx context.Context, msg []byte) error {
	return pubsub.Topic{Client: t.Client}.Publish(ctx, msg)
}

func (t Topic) Subscribe(ctx context.Context) (Subscription, error) {
	out := make(chan []byte, 32)

	cancel, err := pubsub.Topic{Client: t.Client}.Subscribe(ctx, out)
	return Subscription{
		name:   t.Name,
		cancel: cancel,
		c:      out,
		done:   t.done,
	}, err

}

type Subscription struct {
	name   string
	cancel func()
	c      <-chan []byte
	done   <-chan struct{} // rpc.Conn.Done()
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
	case b := <-s.c:
		return b, nil

	case <-ctx.Done():
		return nil, ctx.Err()

	case <-s.done:
		// Cluster connection was lost, but we may still have
		// messages buffered in the channel.
	}

	// Consume remaining messages before returning error.
	select {
	case b := <-s.c:
		return b, nil

	case <-ctx.Done():
		return nil, ctx.Err()

	default:
		return nil, ErrDisconnected
	}
}
