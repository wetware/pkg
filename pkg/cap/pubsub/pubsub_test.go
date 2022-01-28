package pubsub_test

import (
	"context"
	"sync"
	"testing"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	syncutil "github.com/lthibault/util/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mx "github.com/wetware/matrix/pkg"
	pscap "github.com/wetware/ww/pkg/cap/pubsub"
)

func TestPubSub(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sim := mx.New(ctx)

	h := sim.MustHost(ctx)

	gs, err := pubsub.NewGossipSub(ctx, h)
	require.NoError(t, err)

	factory := pscap.New(gs)
	defer factory.Close()

	ps := factory.New(nil)
	defer ps.Release()

	f, release := ps.Join(ctx, "test")
	defer release()

	sub := f.Topic().Subscribe()
	require.NotNil(t, sub, "should always return non-nil subscription")
	defer sub.Cancel()

	err = sub.Resolve(ctx)
	require.NoError(t, err, "should resolve successfully")

	err = f.Topic().Publish(ctx, []byte("test"))
	assert.NoError(t, err, "publish should succeed")

	b, err := sub.Next(ctx)
	require.NoError(t, err, "Next() should succeed")
	assert.Equal(t, "test", string(b), "message should contain 'test'")
}

func TestPubSub_concurrent(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sim := mx.New(ctx)

	h := sim.MustHost(ctx)

	gs, err := pubsub.NewFloodSub(ctx, h)
	require.NoError(t, err)

	f := pscap.New(gs)
	defer f.Close()

	const n = 32
	var wg sync.WaitGroup
	var b = syncutil.NewBarrierChan(n)
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()

			ps := f.New(nil)
			defer ps.Release()

			f, release := ps.Join(ctx, "test")
			defer release()

			sub := f.Topic().Subscribe()
			assert.NotNil(t, sub, "should always return non-nil subscription")
			defer sub.Cancel()

			err := sub.Resolve(ctx)
			assert.NoError(t, err, "should resolve successfully")

			// Wait for all goroutines to resolve their subscriptions,
			// so that nobody misses a message.
			b.SignalAndWait(func() {
				t.Log("all topics resolved")
			})

			err = f.Topic().Publish(ctx, []byte("test"))
			assert.NoError(t, err, "publish should succeed")

			b, err := sub.Next(ctx)
			assert.NoError(t, err, "Next() should succeed")
			assert.Equal(t, "test", string(b), "message should contain 'test'")
		}()
	}

	wg.Wait()
}
