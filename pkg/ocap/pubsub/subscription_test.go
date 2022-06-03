package pubsub

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	chan_api "github.com/wetware/ww/internal/api/channel"
	api "github.com/wetware/ww/internal/api/pubsub"
	"github.com/wetware/ww/pkg/ocap/channel"
)

func TestSubscription_refcount(t *testing.T) {
	t.Parallel()
	t.Helper()

	/*
	 *  Checks that releasing a Topic causes all handlers to be
	 *  released, and that this in turn closes the subscription
	 *  and its channel.
	 */

	t.Run("HandlerClosesChannel", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		topic := Topic(api.Topic_ServerToClient(mockTopic{}, nil))
		defer topic.Release()

		ctx = context.WithValue(ctx, keySenderCallback{},
			senderCallback(func(ch chan_api.Sender) error {
				// Release the channel AFTER we have written a message to the
				// subscription channel.
				defer ch.Release()

				f, release := ch.Send(ctx, channel.Data([]byte("hello, world")))
				defer release()

				_, err := f.Struct()
				return err
			}))

		ch := make(chan []byte, 1)

		release, err := topic.Subscribe(ctx, ch)
		require.NoError(t, err)
		defer release()

		// Ensure we have a message in the subscription channel. The
		// channel should have already been closed, but we should be
		// able to read the buffered message.
		require.Len(t, ch, 1,
			"message should be buffered in subscription channel")

		assert.Equal(t, []byte("hello, world"), <-ch,
			"should receive message")

		_, ok := <-ch
		require.False(t, ok, "channel should be closed")
	})

	t.Run("TopicReleasesHandler", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		topic := Topic(api.Topic_ServerToClient(mockTopic{}, nil))
		defer topic.Release()

		ctx = context.WithValue(ctx, keySenderCallback{},
			senderCallback(func(ch chan_api.Sender) error {
				f, release := ch.Send(ctx, channel.Data([]byte("hello, world")))
				defer release()

				_, err := f.Struct()
				return err
			}))

		ch := make(chan []byte, 1)

		release, err := topic.Subscribe(ctx, ch)
		require.NoError(t, err)
		defer release()

		// Release the topic AFTER we have written a message to the
		// subscription channel.
		topic.Release()

		// Ensure we have a message in the subscription channel. The
		// channel should have already been closed, but we should be
		// able to read the buffered message.
		require.Len(t, ch, 1,
			"message should be buffered in subscription channel")

		assert.Equal(t, []byte("hello, world"), <-ch,
			"should receive message")

		_, ok := <-ch
		require.False(t, ok, "channel should be closed")
	})
}

type mockTopic struct{}

func (mockTopic) Publish(ctx context.Context, call api.Topic_publish) error {
	handle := ctx.Value(keyPublishCallback{}).(publishCallback)
	return handle(call.Args())
}

func (mockTopic) Subscribe(ctx context.Context, call api.Topic_subscribe) error {
	handle := ctx.Value(keySenderCallback{}).(senderCallback)
	return handle(call.Args().Chan())
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

type senderCallback func(chan_api.Sender) error
type keySenderCallback struct{}

type keyName struct{}
