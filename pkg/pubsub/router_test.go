package pubsub_test

import (
	"context"
	"testing"

	capnp "capnproto.org/go/capnp/v3"
	"github.com/golang/mock/gomock"
	logtest "github.com/lthibault/log/test"
	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/pkg/pubsub"
)

func init() {
	capnp.SetClientLeakFunc(func(msg string) {
		panic(msg)
	})
}

func TestRouter(t *testing.T) {
	t.Parallel()

	const name = "test"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := logtest.NewMockLogger(ctrl)
	logger.EXPECT().
		WithField("topic", name).
		Return(logger).
		Times(1)

	ps, release := newGossipSub(ctx)
	defer release()

	r := &pubsub.Server{
		Log:         logger,
		TopicJoiner: ps,
	}

	joiner := r.PubSub()
	defer joiner.Release()

	topic, release := joiner.Join(ctx, name)
	defer release()
	require.NotZero(t, topic, "should not return null capability")

	err := capnp.Client(topic).Resolve(ctx)
	require.NoError(t, err, "should resolve topic capability")
	require.True(t, capnp.Client(topic).IsValid(), "client should be valid")
}
