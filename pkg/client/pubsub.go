package client

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"github.com/ipfs/go-log"
	"github.com/wetware/ww/pkg/cap/pubsub"
)

type Topic interface {
	String() string
	log.Loggable
	Release()
	Publish(context.Context, []byte) error
	Subscribe(context.Context) (Subscription, error)
}

type Subscription interface {
	String() string
	log.Loggable
	Out() <-chan []byte
	Cancel()
}

type futureTopic struct {
	name    string
	f       pubsub.FutureTopic
	release capnp.ReleaseFunc
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
		topic:  t,
		c:      out,
		cancel: cancel,
	}, err

}

type subscription struct {
	topic  *futureTopic
	c      <-chan []byte
	cancel func()
}

func (s *subscription) Out() <-chan []byte { return s.c }
func (s *subscription) Cancel()            { s.cancel() }
func (s *subscription) String() string     { return s.topic.name }

func (s *subscription) Loggable() map[string]interface{} {
	return s.topic.Loggable()
}
