package pubsub_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	logtest "github.com/lthibault/log/test"
	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/pkg/pubsub"
)

func TestRouter(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := logtest.NewMockLogger(ctrl)
	logger.EXPECT().
		WithField("topic", gomock.Any()).
		Times(1)

	ps, release := newGossipSub(ctx)
	defer release()

	r := &pubsub.Router{
		Log:         logger,
		TopicJoiner: ps,
	}

	joiner := r.PubSub()
	defer joiner.Release()

	topic, release := joiner.Join(ctx, "test")
	defer release()
	require.NotZero(t, topic, "should not return null capability")
}
