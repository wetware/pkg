package anchor

import (
	"context"
	"testing"
	"time"

	"capnproto.org/go/capnp/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLs(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()

		s := server{Node: new(Node)}

		it, release := Anchor(s.Anchor()).Ls(context.Background())
		defer release()

		for name := it.Next(); name != ""; name = it.Next() {
			t.Error("iterator should be empty")
		}

		assert.Zero(t, it.Anchor(), "should return null Anchor")
		assert.NoError(t, it.Err(), "iterator should succeed")
	})

	t.Run("NotEmpty", func(t *testing.T) {
		t.Parallel()

		// Create a few children of the root node.
		// These will be cleaned up when the test
		// finishes.
		s := server{Node: new(Node)}
		defer s.Child("foo").AddRef().Release()
		defer s.Child("bar").AddRef().Release()
		defer s.Child("baz").AddRef().Release()

		it, release := Anchor(s.Anchor()).Ls(context.Background())
		defer release()

		var names []string
		var anchors []Anchor
		for name := it.Next(); name != ""; name = it.Next() {
			names = append(names, name)
			anchors = append(anchors, it.Anchor())
		}

		assert.ElementsMatch(t, []string{"foo", "bar", "baz"}, names)
		assert.Len(t, anchors, 3, "should have three anchors")
		for _, anchor := range anchors {
			assert.NotZero(t, anchor, "should return non-null anchor")
		}
	})
}

func TestWalk(t *testing.T) {
	t.Parallel()
	t.Helper()

	ctx := context.Background()

	t.Run("Root", func(t *testing.T) {
		t.Parallel()

		s := server{Node: new(Node)}

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

		s := server{Node: new(Node)}

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

		s := server{Node: new(Node)}

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
