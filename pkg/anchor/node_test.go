package anchor

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"capnproto.org/go/capnp/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeakClient(t *testing.T) {
	t.Parallel()

	client := capnp.ErrorClient(errors.New("test")) // non-null client
	wc := (*weakClient)(client.WeakRef())
	assert.True(t, client.IsSame(wc.AddRef()),
		"should return strong reference to underlying *WeakClient")

	wc = (*weakClient)(capnp.Client{}.WeakRef())
	assert.Panics(t, func() { wc.AddRef() },
		"should panic when creating reference to null *WeakClient")
}

func TestNodeRelease(t *testing.T) {
	t.Parallel()

	n := mknode(func() {}).AddRef()
	n.Release()
	assert.Panics(t, n.Release, "should panic when refcount < 0")
}

func TestNode_Child(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("CreateOne", func(t *testing.T) {
		t.Parallel()

		var released bool
		n := mknode(func() { released = true })

		// note the call to AddRef, nodes initially have a ref-
		// count of zero.
		u := n.Child("child").AddRef()
		require.NotEqual(t, n, u,
			"child should not be root anchor")
		assert.Equal(t, int32(1), n.refs.Load(),
			"child should create a reference to root node")

		u.Release()
		assert.True(t, released, "child should steal parent's reference")
		assert.Empty(t, n.children, "should prune children")
	})

	t.Run("CreateMany", func(t *testing.T) {
		t.Parallel()

		var released bool
		n := mknode(func() { released = true })

		// note the call to AddRef, nodes initially have a ref-
		// count of zero.
		u := n.Child("foo").AddRef()
		require.NotEqual(t, n, u,
			"child should not be root anchor")
		require.Contains(t, n.children, "foo",
			"root should contain child 'foo'")

		// note the call to AddRef, nodes initially have a ref-
		// count of zero.
		u2 := n.Child("bar").AddRef()
		require.NotEqual(t, n, u2,
			"child should not be root anchor")
		require.Contains(t, n.children, "bar",
			"root should contain child 'bar'")

		u.Release()
		require.False(t, released,
			"second child should keep parent alive")
		assert.NotContains(t, n.children, "foo",
			"should remove child 'foo' from parent when releasing")

		u2.Release()
		assert.True(t, released,
			"should release parent after second child reference is released")
		assert.NotContains(t, n.children, "bar",
			"should remove child 'bar' from parent when releasing")

		assert.Empty(t, n.children, "should prune children")
	})

	t.Run("Chain", func(t *testing.T) {
		t.Parallel()

		var released bool
		n := mknode(func() { released = true })

		// note the call to AddRef, nodes initially have a ref-
		// count of zero.
		u := n.Child("foo").Child("bar").AddRef()
		require.False(t, released, "should not release root")
		require.Contains(t, n.children, "foo",
			"should contain child 'foo'")

		u.Release()
		assert.True(t, released, "should release full path")
		assert.NotContains(t, n.children, "foo",
			"should remove child 'foo' from parent when releasing")

		assert.Empty(t, n.children, "should prune children")
	})

	t.Run("Tree", func(t *testing.T) {
		t.Parallel()

		var released bool
		n := mknode(func() { released = true })

		// note the call to AddRef, nodes initially have a ref-
		// count of zero.
		u := n.Child("foo").Child("bar").Child("baz").AddRef()
		require.False(t, released, "should not release root")
		require.Contains(t, n.children, "foo",
			"should contain child 'foo'")

		// note the call to AddRef, nodes initially have a ref-
		// count of zero.
		u2 := n.Child("foo").Child("quxx").AddRef()
		require.False(t, released, "should not release root")

		u.Release()
		require.False(t, released,
			"second path should keep root alive")
		require.Contains(t, n.children, "foo",
			"node 'quxx' should keep node 'foo' alive")

		u2.Release()
		assert.True(t, released, "should release full path")
		assert.NotContains(t, n.children, "bar",
			"should remove child 'bar' from parent when releasing")

		assert.Empty(t, n.children, "should prune children")
	})
}

func TestNode_Anchor(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("Create", func(t *testing.T) {
		t.Parallel()

		var released atomic.Bool
		n := mknode(func() { released.Store(true) })

		a := n.Anchor()
		require.NotZero(t, a, "should return non-nil client")
		require.False(t, released.Load(),
			"should not release before client")

		a.Release()
		assert.Eventually(t, released.Load, time.Second, time.Millisecond*10,
			"should release after client")
	})

	t.Run("Exists", func(t *testing.T) {
		t.Parallel()

		var released atomic.Bool
		n := mknode(func() { released.Store(true) })

		t.Log(n.refs.Load())

		a := n.Anchor()
		require.NotZero(t, a, "should return non-nil client")
		require.False(t, released.Load(),
			"should not release before client")

		t.Log(n.refs.Load())

		b := n.Anchor()
		require.NotZero(t, b, "should return non-nil client")
		require.False(t, released.Load(),
			"should not release before client")

		a.Release()
		require.False(t, released.Load(),
			"should not release before client")

		b.Release()
		assert.Eventually(t, released.Load, time.Second, time.Millisecond*10,
			"should release after client")
	})
}
