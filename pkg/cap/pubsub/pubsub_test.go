package pubsub_test

import (
	"context"
	"testing"
	"time"

	goprocessctx "github.com/jbenet/goprocess/context"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	syncutil "github.com/lthibault/util/sync"
	"golang.org/x/sync/errgroup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wetware/casm/pkg/boot"
	mx "github.com/wetware/matrix/pkg"
	pscap "github.com/wetware/ww/pkg/cap/pubsub"
)

func TestPubSub(t *testing.T) {
	t.Parallel()

	const n = 1 // XXX - set increase to publish concurrently

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sim := mx.New(ctx)

	ps0, ps1 := mustPubSubPair(ctx, sim)

	f0 := pscap.New(ps0)
	f1 := pscap.New(ps1)

	proc := goprocessctx.WithContext(ctx)
	proc.Go(f0.Run())
	proc.Go(f1.Run())

	ft0, release := f0.New(nil).Join(ctx, "test")
	defer release()

	sub, err := ft0.Topic().Subscribe(ctx)
	require.NoError(t, err, "subscription should succeed")

	ft1, release := f1.New(nil).Join(ctx, "test")
	defer release()

	// Test pubsub
	var g errgroup.Group

	for i := 0; i < n; i++ {
		g.Go(func() error {
			err := ft1.Topic().Publish(ctx, []byte("hello, world!"))
			return err
		})
	}

	for i := 0; i < n; i++ {
		b, err := sub.Next(ctx)
		require.NoError(t, err, "should receive message %d", i)
		assert.Equal(t, "hello, world!", string(b))
	}

	require.NoError(t, g.Wait(),
		"should complete %d concurrent rounds of pubsub", n)

	// Subscription should return an error when canceled.
	require.NotPanics(t, func() {
		var g syncutil.FuncGroup
		for i := 0; i < 1000; i++ {
			g.Go(sub.Cancel)
		}
		g.Wait()
	}, "cancel shoud be idempotent and thread-safe")

	b, err := sub.Next(ctx)
	require.Error(t, err,
		"should receive ErrClosed after cancelling subscription")
	require.Nil(t, b,
		"should receive nil message after cancelling subscription")
}

func mustPubSubPair(ctx context.Context, sim mx.Simulation) (ps0, ps1 *pubsub.PubSub) {
	h0 := sim.MustHost(ctx)
	h1 := sim.MustHost(ctx)

	var err error
	ps0, err = pubsub.NewGossipSub(ctx, h0, pubsub.WithDiscovery(hostDiscovery{h1}))
	if err != nil {
		panic(err)
	}

	ps1, err = pubsub.NewGossipSub(ctx, h1, pubsub.WithDiscovery(hostDiscovery{h0}))
	if err != nil {
		panic(err)
	}

	return
}

type hostDiscovery struct{ host.Host }

func (hostDiscovery) Advertise(context.Context, string, ...discovery.Option) (time.Duration, error) {
	return time.Hour, nil
}

func (h hostDiscovery) FindPeers(context.Context, string, ...discovery.Option) (<-chan peer.AddrInfo, error) {
	return boot.StaticAddrs{*host.InfoFromHost(h)}.FindPeers(context.Background(), "")
}
