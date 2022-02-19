package pubsub

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	api "github.com/wetware/ww/internal/api/pubsub"
)

func TestHandler(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("Handle", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ms := make(chan []byte, 1)
		h := handler{
			ms:      ms,
			release: func() {},
		}

		c := api.Topic_Handler_ServerToClient(h, nil)
		defer c.Release()

		f, release := c.Handle(ctx, func(ps api.Topic_Handler_handle_Params) error {
			return ps.SetMsg([]byte("test"))
		})
		defer release()

		_, err := f.Struct()
		assert.NoError(t, err, "call to Handle should succeed")
		assert.Equal(t, "test", string(<-ms), "unexpected message")
	})

	t.Run("Release", func(t *testing.T) {
		t.Parallel()

		var (
			called bool
			ms     = make(chan []byte, 1)
		)

		h := handler{
			ms:      ms,
			release: func() { called = true },
		}

		c := api.Topic_Handler_ServerToClient(h, nil)
		c.Release()

		require.True(t, called, "should call release function")

		_, ok := <-ms
		assert.False(t, ok, "should close message channel when released")
	})
}

// func TestSubscription(t *testing.T) {
// 	t.Parallel()

// 	// ctx, cancel := context.WithCancel(context.Background())
// 	// defer cancel()

// 	// var called bool
// 	// topic := api.Topic_ServerToClient(mockTopicServer(func() { called = true }), nil)
// 	// defer func() {
// 	// 	topic.Release()
// 	// 	require.True(t, called, "should release topic when canceling sub")
// 	// }()

// 	// ms := make(chan []byte, 1)
// 	// ms <- []byte("test")

// 	// ch := make(chan []byte, 1)

// 	// sub, err := newSubscription(ctx, topic, ms)
// 	// require.NoError(t, err, "should create new subscription")
// 	// require.NotNil(t, sub, "should return a subscription client")

// 	// t.Run("Next", func(t *testing.T) {
// 	// 	b, err := sub.Next(ctx)
// 	// 	require.NoError(t, err, "should return message")
// 	// 	require.Equal(t, "test", string(b))
// 	// })

// 	// t.Run("NextWithCanceledCtx", func(t *testing.T) {
// 	// 	ctx, cancel := context.WithCancel(ctx)
// 	// 	cancel()
// 	// 	_, err = sub.Next(ctx)
// 	// 	require.ErrorIs(t, err, context.Canceled, "should abort")
// 	// })

// 	// t.Run("NextWithCanceledSub", func(t *testing.T) {
// 	// 	sub.Cancel()
// 	// 	_, err = sub.Next(ctx)
// 	// 	require.ErrorIs(t, err, ErrClosed, "should be closed")
// 	// })
// }

// // type mockTopicServer func()

// // func (f mockTopicServer) Shutdown() { f() }

// // func (mockTopicServer) Publish(context.Context, api.Topic_publish) error {
// // 	panic("NOT IMPLEMENTED")
// // }

// // func (ch mockTopicServer) Subscribe(context.Context, api.Topic_subscribe) error {
// // 	return nil
// // }
