package pubsub

import (
	"context"
	"sync"
	"testing"

	capnp "capnproto.org/go/capnp/v3"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	inproc "github.com/lthibault/go-libp2p-inproc-transport"
	logtest "github.com/lthibault/log/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	api "github.com/wetware/ww/internal/api/pubsub"
)

func TestTopicManager(t *testing.T) {
	t.Parallel()
	t.Helper()

	const name = "test"

	t.Run("CreateTopic", func(t *testing.T) {
		t.Parallel()

		/*
			Acquire and immediately release.
		*/

		var manager topicManager

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		logger := logtest.NewMockLogger(ctrl)
		logger.EXPECT().
			WithField("topic", name).
			Return(logger).
			Times(1)
		logger.EXPECT().
			Debug("joined topic").
			Times(1)
		logger.EXPECT().
			Debug("left topic").
			Times(1)

		joiner, release := newGossipSub(ctx)
		defer release()

		topic, err := manager.GetOrCreate(ctx, logger, joiner, name)
		require.NoError(t, err, "should create new topic")
		defer topic.Release()

		err = capnp.Client(topic).Resolve(ctx)
		require.NoError(t, err, "should resolve")
		require.True(t, capnp.Client(topic).IsValid(), "client should be valid")
	})

	t.Run("GetTopic", func(t *testing.T) {
		t.Parallel()

		const n = 32
		var manager topicManager

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		logger := logtest.NewMockLogger(ctrl)
		logger.EXPECT().
			WithField("topic", name).
			Return(logger).
			Times(n)
		logger.EXPECT().
			Debug("joined topic").
			Times(1)
		logger.EXPECT().
			Trace("topic ref acquired").
			Times(n - 1)
		logger.EXPECT().
			Debug("left topic").
			Times(1)

		joiner, release := newGossipSub(ctx)
		defer release()

		var ts []api.Topic
		for i := 0; i < n; i++ {
			topic, err := manager.GetOrCreate(ctx, logger, joiner, name)
			require.NoError(t, err, "should get existing topic")
			defer topic.Release()
			ts = append(ts, topic)
		}

		for _, topic := range ts {
			err := capnp.Client(topic).Resolve(ctx)
			require.NoError(t, err, "should resolve topic")
			require.NotPanics(t, topic.Release, "should release topic")
		}
	})

	t.Run("TopicRefsAreIndependent", func(t *testing.T) {
		/*
			Checks an edge-case wherein releasing the first api.Topic
			will cause subsequent calls to GetOrCreate to panic. This
			happens because the first api.Topic is used to derive all
			future references.
		*/

		const n = 3
		var manager topicManager

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		logger := logtest.NewMockLogger(ctrl)
		logger.EXPECT().
			WithField("topic", name).
			Return(logger).
			Times(n)
		logger.EXPECT().
			Debug("joined topic").
			Times(1)
		logger.EXPECT().
			Trace("topic ref acquired").
			Times(n - 1)
		logger.EXPECT().
			Debug("left topic").
			Times(1)

		joiner, release := newGossipSub(ctx)
		defer release()

		topic, err := manager.GetOrCreate(ctx, logger, joiner, name)
		require.NoError(t, err, "should create new topic")
		defer topic.Release()

		// create a second client to ensure the topic stays alive
		t2, err := manager.GetOrCreate(ctx, logger, joiner, name)
		require.NoError(t, err, "should get existing topic")
		defer t2.Release()

		// release the initial client
		require.NotPanics(t, topic.Release,
			"should release initial topic")

		// check that we can still get a reference to the topic
		require.NotPanics(t, func() {
			t3, err := manager.GetOrCreate(ctx, logger, joiner, name)
			assert.NoError(t, err, "should get existing topic")
			t3.Release()
		})
	})
}

func TestManagedServer(t *testing.T) {
	t.Parallel()
	t.Helper()

	var h mockClientHook

	var s = &managedServer{
		refs:       2,
		mu:         new(sync.Mutex),
		ClientHook: &h,
	}

	t.Run("NoShutdownIfRefs", func(t *testing.T) {
		require.NotPanics(t, s.Shutdown,
			"should not panic during shutdown")
		assert.Equal(t, 1, s.refs,
			"should have decremented references by one")
		assert.False(t, bool(h),
			"should not have triggered shutdown")
	})

	t.Run("ShutdownWhenReleased", func(t *testing.T) {
		require.NotPanics(t, s.Shutdown,
			"should not panic during shutdown")
		assert.Zero(t, s.refs,
			"should have decremented references by one")
		assert.True(t, bool(h),
			"should not have triggered shutdown")
	})

	t.Run("InvalidRefPanics", func(t *testing.T) {
		assert.Panics(t, s.Shutdown,
			"should panic if refcount is zero")
	})
}

func newGossipSub(ctx context.Context) (*pubsub.PubSub, func()) {
	h := newTestHost()

	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		panic(err)
	}

	return ps, func() { h.Close() }
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

type mockClientHook bool

func (mockClientHook) Send(context.Context, capnp.Send) (*capnp.Answer, capnp.ReleaseFunc) {
	panic("NOT IMPLEMENTED")
}

func (mockClientHook) Recv(context.Context, capnp.Recv) capnp.PipelineCaller {
	panic("NOT IMPLEMENTED")
}

func (mockClientHook) Brand() capnp.Brand {
	panic("NOT IMPLEMENTED")
}

func (hook *mockClientHook) Shutdown() {
	*hook = true
}
