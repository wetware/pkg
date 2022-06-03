package client_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	chan_api "github.com/wetware/ww/internal/api/channel"
	api "github.com/wetware/ww/internal/api/pubsub"
	"github.com/wetware/ww/pkg/client"
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

		topic := client.NewTopic(
			api.Topic_ServerToClient(mockTopic{}, nil).Client,
			"")
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

		sub, err := topic.Subscribe(ctx)
		require.NoError(t, err, "should subscribe successfully")
		defer sub.Cancel()

		// The channel should have already been closed, but we should be
		// able to read the buffered message.
		b, err := sub.Next(ctx)
		require.NoError(t, err, "should receive message")
		assert.Equal(t, []byte("hello, world"), b, "should receive expected message")

		b, err = sub.Next(ctx)
		require.ErrorIs(t, err, client.ErrDisconnected, "subscription should be closed")
		require.Nil(t, b, "should not return data")
	})

	t.Run("TopicReleasesHandler", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		topic := client.NewTopic(
			api.Topic_ServerToClient(mockTopic{}, nil).Client,
			"")
		defer topic.Release()

		ctx = context.WithValue(ctx, keySenderCallback{},
			senderCallback(func(ch chan_api.Sender) error {
				f, release := ch.Send(ctx, channel.Data([]byte("hello, world")))
				defer release()

				_, err := f.Struct()
				return err
			}))

		sub, err := topic.Subscribe(ctx)
		require.NoError(t, err, "should subscribe successfully")
		defer sub.Cancel()

		// Release the topic AFTER we have written a message to the
		// subscription channel.
		topic.Release()

		// The handler should have already been closed, but we should be
		// able to read the buffered message.
		b, err := sub.Next(ctx)
		require.NoError(t, err, "should receive message")
		assert.Equal(t, []byte("hello, world"), b, "should receive expected message")

		b, err = sub.Next(ctx)
		require.ErrorIs(t, err, client.ErrDisconnected, "subscription should be closed")
		require.Nil(t, b, "should not return data")
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
