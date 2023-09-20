package system_test

import (
	"context"
	"testing"

	rpccp "capnproto.org/go/capnp/v3/std/capnp/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wetware/pkg/system"
	"zenhack.net/go/util/rc"
)

func TestPipe(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()

		pipe := system.NewPipe()

		want := new(rc.Ref[rpccp.Message])
		defer want.Release()

		got, err := pipe.Pop(context.TODO())
		require.ErrorIs(t, err, context.DeadlineExceeded, "should signal empty queue")
		assert.Nil(t, got, "should return nil reference")

		err = pipe.Push(context.TODO(), want)
		require.NoError(t, err, "push should succeed")

		got, err = pipe.Pop(context.TODO())
		require.NoError(t, err, "pop should succeed")
		require.Equal(t, want, got, "should get expected item from the queue")
	})

	t.Run("Full", func(t *testing.T) {
		t.Parallel()

		pipe := system.NewPipe()

		want := new(rc.Ref[rpccp.Message])
		defer want.Release()

		err := pipe.Push(context.TODO(), want)
		require.NoError(t, err, "push should succeed")

		err = pipe.Push(context.TODO(), want)
		require.ErrorIs(t, err, context.DeadlineExceeded,
			"should signal that buffer is full")
	})

	t.Run("Closed", func(t *testing.T) {
		t.Parallel()

		pipe := system.NewPipe()

		err := pipe.Close()
		require.NoError(t, err, "should close successfully")

		err = pipe.Push(context.TODO(), nil)
		require.EqualError(t, err, "closed", "pipe should be closed")

		ref, err := pipe.Pop(context.TODO())
		require.EqualError(t, err, "closed", "pipe should be closed")
		require.Nil(t, ref, "should not return ref")
	})
}
