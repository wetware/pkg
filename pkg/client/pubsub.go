package client

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"github.com/ipfs/go-log"
	"github.com/wetware/ww/pkg/cap/pubsub"
)

type Topic interface {
	log.Loggable
	String() string
	Publish(context.Context, []byte) error
	Subscribe(context.Context) (Subscription, error)
	Release()
}

type Subscription interface {
	log.Loggable
	String() string
	Next(context.Context) ([]byte, error)
	Cancel()
}

type futureTopic struct {
	name    string
	f       pubsub.FutureTopic
	release capnp.ReleaseFunc
	done    <-chan struct{} // rpc.Conn.Done()
}

func (t *futureTopic) String() string { return t.name }

func (t *futureTopic) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"topic": t.name,
	}
}

func (t *futureTopic) Release() { t.release() }

func (t *futureTopic) Publish(ctx context.Context, msg []byte) error {
	return t.f.Topic().Publish(ctx, msg)
}

func (t *futureTopic) Subscribe(ctx context.Context) (Subscription, error) {
	topic, err := t.f.Struct()
	if err != nil {
		return nil, err
	}

	out := make(chan []byte, 32)

	cancel, err := topic.Subscribe(ctx, out)
	return &subscription{
		done:   t.done,
		topic:  t,
		c:      out,
		cancel: cancel,
	}, err

}

type subscription struct {
	done   <-chan struct{} // rpc.Conn.Done()
	topic  *futureTopic
	c      <-chan []byte
	cancel func()
}

func (s *subscription) Cancel()        { s.cancel() }
func (s *subscription) String() string { return s.topic.name }

func (s *subscription) Loggable() map[string]interface{} {
	return s.topic.Loggable()
}

func (s *subscription) Next(ctx context.Context) ([]byte, error) {
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
