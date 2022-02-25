package ww_test

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
	"github.com/wetware/casm/pkg/boot"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/client"
	"github.com/wetware/ww/pkg/server"
	"github.com/wetware/ww/pkg/vat"
)

func TestProto(t *testing.T) {
	t.Parallel()

	const ns = "test"
	match := ww.NewMatcher(ns)
	proto := ww.Subprotocol(ns)
	t.Log(proto)

	assert.True(t, match(string(proto)),
		"matcher should match subprotocol")
}

func TestClientServer_integration(t *testing.T) {
	t.Parallel()

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h, err := libp2p.New(ctx,
		libp2p.NoListenAddrs,
		libp2p.ListenAddrStrings("/inproc/~"),
		libp2p.NoTransports,
		libp2p.Transport(inproc.New()))
	require.NoError(t, err, "should spawn server host")

	ps, err := pubsub.NewGossipSub(ctx, h)
	require.NoError(t, err, "should create gossipsub")

	sn, err := server.New(ctx, vat.Network{NS: "test", Host: h}, ps,
		server.WithLogger(log))
	require.NoError(t, err, "should spawn server")
	defer func() {
		assert.NoError(t, sn.Close(), "server should close gracefully")
	}()

	h2, err := libp2p.New(ctx,
		libp2p.NoListenAddrs,
		libp2p.NoTransports,
		libp2p.Transport(inproc.New()))
	require.NoError(t, err, "must succeed")

	cn, err := client.Dialer{
		Boot: boot.StaticAddrs{*host.InfoFromHost(h)},
		Vat:  vat.Network{NS: "test", Host: h2},
	}.Dial(ctx)
	require.NoError(t, err, "should dial cluster")
	defer func() {
		assert.NoError(t, cn.Close(), "client should close gracefully")
	}()

	t.Run("PubSub", func(t *testing.T) {
		const topic = "test.pubsub.send_recv"

		top := cn.Join(ctx, topic)
		defer top.Release()

		sub, err := top.Subscribe(ctx)
		require.NoError(t, err, "should subscribe successfully")
		defer sub.Cancel()

		err = top.Publish(ctx, []byte("hello, world!"))
		require.NoError(t, err, "should publish message")

		require.Eventually(t, func() bool {
			return len(sub.C) > 0
		}, time.Millisecond*10, time.Millisecond, "should receive a message")

		assert.Equal(t, "hello, world!", string(<-sub.C),
			"should match previously-published message")
	})
}
