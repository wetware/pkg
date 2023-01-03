package pubsub_test

import (
	"context"
	"testing"
	"time"

	"capnproto.org/go/capnp/v3/exc"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mock_pubsub "github.com/wetware/ww/internal/mock/pkg/pubsub"
	"github.com/wetware/ww/pkg/pubsub"
)

func TestNullSubscription(t *testing.T) {
	t.Parallel()

	topic, release := pubsub.Topic{}.Subscribe(context.Background())
	defer release()

	assert.Nil(t, topic.Next(), "should be exhausted")

	require.NotNil(t, topic.Err(), "should return error")

	failed := exc.IsType(topic.Err(), exc.Failed)
	assert.True(t, failed, "should return 'failed' exception")
}

func TestSubscribe_cancel(t *testing.T) {
	t.Parallel()

	/*
		Test that releasing a subscription causes the iterator to
		unblock.
	*/

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server := mock_pubsub.NewMockTopicServer(ctrl)
	server.EXPECT().
		Subscribe(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, _ pubsub.MethodSubscribe) error {
			<-ctx.Done()
			return ctx.Err()
		}).
		Times(1)

	topic := pubsub.NewTopic(server)
	defer topic.Release()

	sub, release := topic.Subscribe(context.Background())
	release() // immediate release

	select {
	case <-sub.Future.Done():
	case <-time.After(time.Millisecond * 100):
		t.Error("should cancel subscription quickly")
	}
}
