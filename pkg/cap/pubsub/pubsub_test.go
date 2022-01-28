package pubsub_test

import (
	"context"
	"testing"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
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
