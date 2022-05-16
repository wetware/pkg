package pubsub

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	api "github.com/wetware/ww/internal/api/pubsub"
)

func TestSubscription_refcount(t *testing.T) {
	t.Parallel()

	/*
	 *  Checks that releasing a Topic causes all handlers to be
	 *  released, and that this in turn closes the subscription
	 *  and its channel.
	 */

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	topic := Topic(api.Topic_ServerToClient(mockTopic{}, nil))
	defer topic.Release()

	ctx = context.WithValue(ctx, keyHandlerCallback{},
		handlerCallback(func(h api.Topic_Handler) error {
			f, release := h.Handle(ctx, func(ps api.Topic_Handler_handle_Params) error {
				return ps.SetMsg([]byte("hello, world"))
			})
			defer release()

			_, err := f.Struct()
			return err
		}))

	ch := make(chan []byte, 1)

	release, err := topic.Subscribe(ctx, ch)
	require.NoError(t, err)
	defer release()

	// Ensure we have a message in the subscription channel, then release
	// the topic.  This MUST cause the subscription channel to be closed,
	// but we should still be able to read the buffered message.
	require.Len(t, ch, 1, "message should be buffered in subscription channel")
	topic.Release()

	var got []byte
	select {
	case got = <-ch:
	default:
	}

	assert.Equal(t, []byte("hello, world"), got,
		"should receive message")

	_, ok := <-ch
	require.False(t, ok, "channel should be closed")
}

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

type mockTopic struct{}

func (mockTopic) Publish(ctx context.Context, call api.Topic_publish) error {
	handle := ctx.Value(keyPublishCallback{}).(publishCallback)
	return handle(call.Args())
}

func (mockTopic) Subscribe(ctx context.Context, call api.Topic_subscribe) error {
	handle := ctx.Value(keyHandlerCallback{}).(handlerCallback)
	return handle(call.Args().Handler())
}

func (mockTopic) Name(ctx context.Context, call api.Topic_name) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetName(ctx.Value(keyName{}).(string))
}

type publishCallback func(args interface{ Msg() ([]byte, error) }) error
type keyPublishCallback struct{}

type handlerCallback func(h api.Topic_Handler) error
type keyHandlerCallback struct{}

type keyName struct{}
