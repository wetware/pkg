package pubsub_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	logtest "github.com/lthibault/log/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mx "github.com/wetware/matrix/pkg"
	pscap "github.com/wetware/ww/pkg/cap/pubsub"
)

func TestPubSub(t *testing.T) {
	t.Parallel()

	const topic = "test"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	log := logtest.NewMockLogger(ctrl)
	log.EXPECT().
		WithField(gomock.Any(), gomock.Any()).
		Return(log).
		AnyTimes()
	log.EXPECT().
		With(gomock.Any()).
		Return(log).
		AnyTimes()
	log.EXPECT().
		Trace(gomock.Any()).
		AnyTimes()
	log.EXPECT().
		Debug(gomock.Any()).
		AnyTimes()

	sim := mx.New(ctx)
	h := sim.MustHost(ctx)

	gs, err := pubsub.NewGossipSub(ctx, h)
	require.NoError(t, err)

	p := pscap.New(gs, pscap.WithLogger(log))
	defer func() {
		assert.NoError(t, p.Close(), "factory should close gracefully")
		assert.ErrorIs(t, p.Close(), pscap.ErrClosed)
	}()

	ps := pscap.PubSub{p.Client()}
	defer ps.Release()

	for i := 0; i < 2; i++ {
		func() {
			f, release := ps.Join(ctx, topic)
			defer release()

			top, err := f.Struct()
			require.NoError(t, err, "should resolve topic")
			defer top.Release()

			sub, err := top.Subscribe(ctx)
			require.NoError(t, err, "should subscribe")
			require.NotNil(t, sub, "should return non-nil subscription")
			defer sub.Cancel()

			err = top.Publish(ctx, []byte("test"))
			require.NoError(t, err, "should publish message")

			b, err := sub.Next(ctx)
			require.NoError(t, err, "should receive message")
			require.NotNil(t, b, "message should contain data")
		}()
	}
}
