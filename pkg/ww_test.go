package ww_test

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	inproc "github.com/lthibault/go-libp2p-inproc-transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wetware/casm/pkg/boot"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/client"
	"github.com/wetware/ww/pkg/server"
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

	sn, err := server.New(ctx, h, ps)
	require.NoError(t, err, "should spawn server")
	defer func() {
		assert.NoError(t, sn.Close(), "server should close gracefully")
	}()

	cn, err := client.DialDiscover(ctx, boot.StaticAddrs{*host.InfoFromHost(h)},
		client.WithHostOpts(
			libp2p.NoListenAddrs,
			libp2p.NoTransports,
			libp2p.Transport(inproc.New())))
	require.NoError(t, err, "should dial cluster")
	defer func() {
		assert.NoError(t, cn.Close(), "client should close gracefully")
	}()

	t.Run("PubSub", func(t *testing.T) {
		const topic = "test"

		f, release := cn.PubSub().Join(ctx, topic)
		defer release()

		top, err := f.Struct()
		require.NoError(t, err, "should resolve topic")
		defer top.Release()

		sub := top.Subscribe()
		defer sub.Cancel()

		time.Sleep(time.Millisecond)

		err = top.Publish(ctx, []byte("hello, world!"))
		require.NoError(t, err, "should publish message")

		b, err := sub.Next(ctx)
		require.NoError(t, err, "should receive message")
		assert.Equal(t, "hello, world!", string(b))
	})
}
