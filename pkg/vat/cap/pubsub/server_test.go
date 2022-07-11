package pubsub_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	inproc "github.com/lthibault/go-libp2p-inproc-transport"
	logtest "github.com/lthibault/log/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pscap "github.com/wetware/ww/pkg/vat/cap/pubsub"
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

	h := newTestHost()
	defer h.Close()

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
			}, time.Second, time.Millisecond, "should receive message")

			assert.Equal(t, "test", string(<-ch),
				"should match previously-published message")
		}()
	}
}

func newTestHost() host.Host {
	h, err := libp2p.New(
		libp2p.NoListenAddrs,
		libp2p.NoTransports,
		libp2p.Transport(inproc.New()),
		libp2p.ListenAddrStrings("/inproc/~"))
	if err != nil {
		panic(err)
	}

	return h
}
