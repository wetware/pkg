package client

// import (
// 	"context"

// 	"github.com/wetware/ww/pkg/pubsub"
// )

// type Topic struct {
// 	Name string
// 	pubsub.Topic
// }

// func (t Topic) String() string { return t.Name }

// func (t Topic) Loggable() map[string]interface{} {
// 	return map[string]interface{}{
// 		"topic": t.Name,
// 	}
// }

// func (t Topic) AddRef() Topic {
// 	return Topic{
// 		Name:  t.Name,
// 		Topic: t.Topic.AddRef(),
// 	}
// }

// func (t Topic) Publish(ctx context.Context, msg []byte) error {
// 	return t.Topic.Publish(ctx, msg)
// }

// func (t Topic) Subscribe(ctx context.Context, opts ...SubOpt) (Subscription, error) {
// 	sub := Subscription{
// 		name: t.Name,
// 		c:    make(chan []byte, 32),
// 	}

// 	for _, option := range opts {
// 		err := option(&sub)
// 		if err != nil {
// 			return sub, err
// 		}
// 	}

// 	release, err := t.Topic.Subscribe(ctx, sub.c)
// 	sub.cancel = release
// 	return sub, err
// }

// type SubOpt func(sub *Subscription) error

// // WithBufferSize is a Subscribe option to customize the size of the subscribe output buffer.
// // The default length is 32 but it can be configured to avoid dropping messages if the consumer is not reading fast
// // enough.
// func WithBufferSize(size int) SubOpt {
// 	return func(sub *Subscription) error {
// 		sub.c = make(chan []byte, size)
// 		return nil
// 	}
// }

// type Subscription struct {
// 	name   string
// 	cancel func()
// 	c      chan []byte
// }

// func (s Subscription) String() string { return s.name }

// func (s Subscription) Loggable() map[string]interface{} {
// 	return map[string]interface{}{
// 		"topic": s.name,
// 	}
// }

// func (s Subscription) Cancel() { s.cancel() }

// func (s Subscription) Next(ctx context.Context) ([]byte, error) {
// 	select {
// 	case b, ok := <-s.c:
// 		if ok {
// 			return b, nil
// 		}

// 		return nil, ErrDisconnected

// 	case <-ctx.Done():
// 		return nil, ctx.Err()
// 	}
// }
