package pubsub_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/wetware/ww/internal/api/pubsub"
	mock_pubsub "github.com/wetware/ww/internal/mock/pkg/pubsub"
	"github.com/wetware/ww/pkg/pubsub"
)

func TestPublish(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server := mock_pubsub.NewMockTopicServer(ctrl)
	server.EXPECT().
		Publish(gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, call api.Topic_publish) {
			b, err := call.Args().Msg()
			assert.NoError(t, err, "should have message argument")
			assert.Equal(t, "hello, world!", string(b),
				"argument should match input")
		}).
		Return(nil).
		Times(1)

	topic := pubsub.NewTopic(server)
	defer topic.Release()

	err := topic.Publish(context.Background(), []byte("hello, world!"))
	require.NoError(t, err, "should succeed")
}

func TestStream(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server := mock_pubsub.NewMockTopicServer(ctrl)
	server.EXPECT().
		Publish(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(128)

	topic := pubsub.NewTopic(server)
	defer topic.Release()

	stream := topic.NewStream(context.Background())
	for i := 0; i < 128; i++ {
		err := stream.Publish([]byte("hello, world!"))
		require.NoError(t, err, "should send text")
	}

	err := stream.Close()
	assert.NoError(t, err, "should close gracefully")
}

func TestSendStream_Error(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	want := errors.New("test")

	server := mock_pubsub.NewMockTopicServer(ctrl)
	server.EXPECT().
		Publish(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(64)
	server.EXPECT().
		Publish(gomock.Any(), gomock.Any()).
		Return(want).
		Times(1)
	server.EXPECT().
		Publish(gomock.Any(), gomock.Any()).
		Return(nil).
		MaxTimes(64)

	topic := pubsub.NewTopic(server)
	defer topic.Release()

	stream := topic.NewStream(context.Background())
	for i := 0; i < 64; i++ {
		err := stream.Publish([]byte("hello, world!"))
		require.NoError(t, err, "should send text")
	}

	// The next call will trigger the error, but it might
	// not be detected synchronously.   We have no way of
	// knowing which of these calls will detect the error.
	for i := 0; i < 64; i++ {
		// Maximize the chance that of detecting the error
		// in-flight.  This helps with code coverage.
		time.Sleep(time.Millisecond)

		err := stream.Publish([]byte("hello, world!"))
		if err != nil {
			require.ErrorIs(t, err, want, "should return error")
		}
	}

	err := stream.Close()
	require.ErrorIs(t, err, want, "should return error")
}
