package ww_test

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	disc "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ctxutil "github.com/lthibault/util/ctx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wetware/casm/pkg/boot"
	mx "github.com/wetware/matrix/pkg"
	"github.com/wetware/ww/pkg/client"
)

func TestClientServer(t *testing.T) {
	t.Parallel()

	const ns = "test"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sim := mx.New(ctx)

	// Set up a host to act as a server.
	// TODO:  replace this with a server.Node at some point ...
	var sr routing.Routing
	serverHost := sim.MustHost(ctx,
		libp2p.ListenAddrStrings("/inproc/server"),
		// Ensure the sever host has a functioning DHT in server mode, which
		// we will also use for pubsub discovery.
		libp2p.Routing(func(h host.Host) (_ routing.PeerRouting, err error) {
			ctx := ctxutil.C(h.Network().Process().Closing())
			sr, err = dual.New(ctx, h, dual.DHTOption(dht.Mode(dht.ModeServer)))
			return sr, err
		}))
	defer serverHost.Close()

	p, err := pubsub.NewGossipSub(ctx, serverHost,
		pubsub.WithDiscovery(disc.NewRoutingDiscovery(sr)))
	require.NoError(t, err)
	require.NotZero(t, p)

	go func() {
		topic, err := p.Join(ns)
		require.NoError(t, err)
		require.NotNil(t, topic)

		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()

		// XXX:  do we need to relay the topic, or is publishing sufficient?

		for {
			select {
			case <-ticker.C:
				err := topic.Publish(ctx, []byte("hello"),
					/*pubsub.WithReadiness(pubsub.MinTopicSize(1))*/)
				require.NoError(t, err)

			case <-ctx.Done():
				return
			}
		}
	}()

	// Create a client host and dial.
	clientHost := sim.MustHost(ctx, libp2p.NoListenAddrs)

	info := *host.InfoFromHost(serverHost)
	c, err := client.DialDiscover(ctx, boot.StaticAddrs{info},
		client.WithHost(clientHost))

	require.NoError(t, err, "client should be able to dial server")
	require.NotZero(t, c, "client should not be zero-valued")
	defer c.Close()

	topic, err := c.PubSub().Join(ns)
	require.NoError(t, err)
	require.NotNil(t, topic)

	sub, err := topic.Subscribe()
	require.NoError(t, err)
	require.NotNil(t, sub)
	defer sub.Cancel()

	recvCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	msg, err := sub.Next(recvCtx)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(msg.GetData()))
}
