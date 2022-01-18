package pubsub

import (
	"context"
	"fmt"
	"testing"
	"time"

	"capnproto.org/go/capnp/v3"
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

		h := newHandler()
		h.ms = make(chan []byte, 1)

		c := api.Topic_Handler_ServerToClient(h, nil)
		f, release := c.Handle(ctx, func(ps api.Topic_Handler_handle_Params) error {
			return ps.SetMsg([]byte("test"))
		})
		defer release()

		_, err := f.Struct()
		assert.NoError(t, err, "call to Handle should succeed")
		assert.Equal(t, "test", string(<-h.ms), "unexpected message")
	})

	t.Run("ConcurrentRelease", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var (
			h     = newHandler()
			cherr = make(chan error, 1)
			sync  = make(chan struct{})
			c     = api.Topic_Handler_ServerToClient(h, nil)
		)

		go func() {
			f, release := c.Handle(ctx, func(ps api.Topic_Handler_handle_Params) error {
				defer close(sync)
				return ps.SetMsg([]byte("test"))
			})
			defer release()

			_, err := f.Struct()
			cherr <- err
		}()

		<-sync
		time.Sleep(time.Millisecond * 10) // ensure the goroutine is blocked
		require.NotPanics(t, h.Shutdown,
			"h.release() should not panic")

		assert.EqualError(t, <-cherr,
			"pubsub.capnp:Topic.Handler.handle: closed",
			"should return ErrClosed from concurrent call to handler")

		// Now that the handler has been released, subsequent calls to Handle
		// should also fail with Errclosed
		f, release := c.Handle(ctx, func(ps api.Topic_Handler_handle_Params) error {
			return ps.SetMsg([]byte("test"))
		})
		defer release()

		_, err := f.Struct()
		assert.EqualError(t, err,
			"pubsub.capnp:Topic.Handler.handle: closed",
			"should return ErrClosed from synchronous call to released handler")
	})
}

func TestSubscription(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("Next", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ch := make(mockTopicServer, 1)
		topic := api.Topic_ServerToClient(ch, nil)
		sub := newSubscription(topic)

		go func() {
			select {
			case <-ctx.Done():
			case sub.h.ms <- []byte("test"):
			}
		}()

		b, err := sub.Next(ctx)
		require.NoError(t, err, "Next() should succeed")
		assert.Equal(t, "test", string(b))
	})

	t.Run("ContextCancel", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ch := make(mockTopicServer, 1)
		topic := api.Topic_ServerToClient(ch, nil)
		sub := newSubscription(topic)

		aborted, abort := context.WithCancel(ctx)
		abort()

		b, err := sub.Next(aborted)
		require.ErrorIs(t, err, context.Canceled)
		assert.Nil(t, b)
	})

	t.Run("Cancel", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ch := make(mockTopicServer, 1)
		topic := api.Topic_ServerToClient(ch, nil)
		sub := newSubscription(topic)

		sub.Cancel()
		require.NotPanics(t, sub.Cancel,
			"sub.Cancel() should be idempotent")

		b, err := sub.Next(ctx)
		require.ErrorIs(t, err, ErrClosed, "should receive ErrClosed")
		require.Nil(t, b, "message should be nil")

		// Try again to check that the resolve function is short-circuited.
		// This is probably a bit pedantic, but it increases test coverage,
		// and also seems to help trigger the <-ctx.Done() condition.
		b, err = sub.Next(ctx)
		require.ErrorIs(t, err, ErrClosed, "should receive ErrClosed")
		require.Nil(t, b, "message should be nil")
	})

	t.Run("HandlerReleased", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ch := make(mockTopicServer, 1)
		topic := api.Topic_ServerToClient(ch, nil)
		sub := newSubscription(topic)

		(<-ch).Release()

		b, err := sub.Next(ctx)
		require.ErrorIs(t, err, ErrClosed, "should receive ErrClosed")
		require.Nil(t, b, "message should be nil")
	})

	t.Run("ResolveFailure", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		topic := api.Topic_ServerToClient(errTopicServer{}, nil)
		sub := newSubscription(topic)

		b, err := sub.Next(ctx)
		require.EqualError(t, err, "pubsub.capnp:Topic.subscribe: test")
		require.Nil(t, b, "message should be nil")
	})
}

type mockTopicServer chan *capnp.Client

func (mockTopicServer) Publish(context.Context, api.Topic_publish) error {
	panic("NOT IMPLEMENTED")
}

func (ch mockTopicServer) Subscribe(ctx context.Context, call api.Topic_subscribe) error {
	ch <- call.Args().Handler().AddRef().Client
	return nil
}

type errTopicServer struct{}

func (errTopicServer) Publish(context.Context, api.Topic_publish) error {
	panic("NOT IMPLEMENTED")
}

func (errTopicServer) Subscribe(ctx context.Context, call api.Topic_subscribe) error {
	return fmt.Errorf("test")
}
