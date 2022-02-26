package pubsub_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	logtest "github.com/lthibault/log/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mx "github.com/wetware/matrix/pkg"
	pscap "github.com/wetware/ww/pkg/cap/pubsub"
)

func TestPubSub_refcount(t *testing.T) {
	t.Parallel()

	const topic = "test"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Use mock logger to detect refcounting errors.  The mock
	// will fail with unexpected calls.
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

	p := pscap.New("test", gs, pscap.WithLogger(log))
	defer func() {
		assert.NoError(t, p.Close(), "factory should close gracefully")
		assert.ErrorIs(t, p.Close(), pscap.ErrClosed)
	}()

	ps := pscap.PubSub{p.Client()}
	defer ps.Release()

	// ensure topic doesn't leak
	for i := 0; i < 2; i++ {
		func() {
			ch := make(chan []byte, 1)

			f, release := ps.Join(ctx, topic)
			defer release()

			top, err := f.Struct()
			require.NoError(t, err, "should resolve topic")
			defer top.Release()

			cancel, err := top.Subscribe(ctx, ch)
			require.NoError(t, err, "should subscribe")
			require.NotNil(t, cancel, "should return a cancellation function")
			defer cancel()

			err = top.Publish(ctx, []byte("test"))
			require.NoError(t, err, "should publish message")

			require.Eventually(t, func() bool {
				return len(ch) == cap(ch)
			}, time.Millisecond*10, time.Millisecond, "should receive message")

			assert.Equal(t, "test", string(<-ch),
				"should match previously-published message")
		}()
	}
}
