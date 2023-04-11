package anchor

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"capnproto.org/go/capnp/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalk(t *testing.T) {
	t.Parallel()
	t.Helper()

	ctx := context.Background()

	t.Run("Root", func(t *testing.T) {
		t.Parallel()

		var released atomic.Bool
		s := server{node: mknode(func() { released.Store(true) })}

		root := Anchor(s.Anchor())

		ref, release := root.Walk(ctx, "/")
		defer release()

		assert.False(t, released.Load(),
			"should not release root node after walk")
		require.True(t, capnp.Client(ref).IsSame(capnp.Client(root)),
			"should return root anchor")

		root.Release()
		assert.False(t, released.Load(),
			"root node should be kept alive by second reference")

		release()
		assert.Eventually(t, released.Load, time.Second, time.Millisecond*10,
			"should release after client")
	})

	t.Run("Child", func(t *testing.T) {
		t.Parallel()

		var released atomic.Bool
		s := server{node: mknode(func() { released.Store(true) })}

		root := Anchor(s.Anchor())

		child, release := root.Walk(ctx, "/foo")
		defer release()

		assert.False(t, released.Load(),
			"should not release root node after walk")
		require.False(t, capnp.Client(child).IsSame(capnp.Client(root)),
			"should return root anchor")

		root.Release()
		assert.False(t, released.Load(),
			"root should be kept alive by child")

		release()
		assert.Eventually(t, released.Load, time.Second, time.Millisecond*10,
			"should release after client")
	})

	t.Run("Path", func(t *testing.T) {
		t.Parallel()

		var released atomic.Bool
		s := server{node: mknode(func() { released.Store(true) })}

		root := Anchor(s.Anchor())

		child, release := root.Walk(ctx, "/foo/bar")
		defer release()

		assert.False(t, released.Load(),
			"should not release root node after walk")
		require.False(t, capnp.Client(child).IsSame(capnp.Client(root)),
			"should return root anchor")

		root.Release()
		assert.False(t, released.Load())

		release()
		assert.Eventually(t, released.Load, time.Second, time.Millisecond*100,
			"should release after client")
	})
}
