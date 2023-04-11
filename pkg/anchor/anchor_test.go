package anchor

import (
	"context"
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

		s := server{node: new(node)}

		root := Anchor(s.Anchor())

		ref, release := root.Walk(ctx, "/")
		defer release()

		assert.NotZero(t, s.refs.Load(),
			"should not release root node after walk")
		require.True(t, capnp.Client(ref).IsSame(capnp.Client(root)),
			"should return root anchor")

		root.Release()
		assert.NotZero(t, s.refs.Load(),
			"root node should be kept alive by second reference")

		release()
		assert.Eventually(t, func() bool {
			return s.refs.Load() == 0
		}, time.Second, time.Millisecond*10,
			"should release after client")
	})

	t.Run("Child", func(t *testing.T) {
		t.Parallel()

		s := server{node: new(node)}

		root := Anchor(s.Anchor())

		child, release := root.Walk(ctx, "/foo")
		defer release()

		assert.NotZero(t, s.refs.Load(),
			"should not release root node after walk")
		require.False(t, capnp.Client(child).IsSame(capnp.Client(root)),
			"should return root anchor")

		root.Release()
		assert.NotZero(t, s.refs.Load(),
			"root should be kept alive by child")

		release()
		assert.Eventually(t, func() bool {
			return s.refs.Load() == 0
		}, time.Second, time.Millisecond*10,
			"should release after client")
	})

	t.Run("Path", func(t *testing.T) {
		t.Parallel()

		s := server{node: new(node)}

		root := Anchor(s.Anchor())

		child, release := root.Walk(ctx, "/foo/bar")
		defer release()

		assert.NotZero(t, s.refs.Load(),
			"should not release root node after walk")
		require.False(t, capnp.Client(child).IsSame(capnp.Client(root)),
			"should return root anchor")

		root.Release()
		assert.NotZero(t, s.refs.Load())

		release()
		assert.Eventually(t, func() bool {
			return s.refs.Load() == 0
		}, time.Second, time.Millisecond*10,
			"should release after client")
	})
}
