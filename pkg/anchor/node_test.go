package anchor

import (
	"errors"
	"testing"
	"time"

	"capnproto.org/go/capnp/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeakClient(t *testing.T) {
	t.Parallel()

	var wc weakClient
	assert.False(t, wc.Exists(), "zero-value weakClient should be null")
	assert.Panics(t, func() { wc.AddRef() }, "should panic on nil *capnp.WeakClient")

	var released bool
	client := capnp.ErrorClient(errors.New("test")) // non-null client
	wc.WeakClient = client.WeakRef()
	wc.release = func() { released = true }
	assert.True(t, wc.Exists(), "*capnp.WeakClient should exist")
	assert.True(t, client.IsSame(wc.AddRef()),
		"should return strong reference to underlying *WeakClient")

	wc.Release()
	assert.True(t, released, "should release refcounter")
	assert.False(t, wc.Exists(), "client should not exist after reset")
}

func TestNode_UseAfterFree(t *testing.T) {
	t.Parallel()

	var released bool
	n := mknode(func() { released = true })

	n.Release()
	assert.True(t, released)

	assert.Nil(t, n.nodestate, "should nil out nodestate after free")
	assert.Panics(t, func() { n.AddRef() }, "should panic if AddRef() is called after free")
	assert.Panics(t, n.Release, "should panic if Release() is called after free")
}

func TestChild(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("Create", func(t *testing.T) {
		t.Parallel()

		var released bool
		n := mknode(func() { released = true })

		u := n.Child("child")
		require.NotEqual(t, n, u, "child should not be root anchor")

		n.Release()
		assert.False(t, released, "child should keep parent alive")

		u.Release()
		assert.True(t, released, "should release parent after releasing child")
	})

	t.Run("Exists", func(t *testing.T) {
		t.Parallel()

		var released bool
		n := mknode(func() { released = true })

		u := n.Child("child")
		require.NotEqual(t, n.nodestate, u.nodestate, "child should not be root anchor")

		u2 := n.Child("child")
		require.NotEqual(t, n.nodestate, u2.nodestate, "child should not be root anchor")
		require.Equal(t, u.nodestate, u2.nodestate, "should be the same child")

		n.Release()
		assert.False(t, released, "child should keep parent alive")

		u.Release()
		assert.False(t, released, "additional child reference should keep parent alive")

		u2.Release()
		assert.True(t, released, "should release parent after second child reference is released")
	})

	t.Run("FreeBeforeChild", func(t *testing.T) {
		t.Parallel()

		var released bool
		n := mknode(func() { released = true })

		u := n.Child("child")
		require.NotEqual(t, n.nodestate, u.nodestate, "child should not be root anchor")

		n.Release()
		assert.False(t, released, "child should keep parent alive")

		// traverse the released node; we want to make sure that we're able to
		// increment refs after they have been released.
		u2 := n.Child("child")
		require.NotEqual(t, n, u, "child should not be root anchor")
		require.Equal(t, u.nodestate, u2.nodestate, "should be the same child")

		u.Release()
		assert.False(t, released, "child should keep parent alive")

		u2.Release()
		assert.True(t, released, "should release parent after releasing child")
	})
}

func TestAnchor(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("Create", func(t *testing.T) {
		t.Parallel()

		var released bool
		n := mknode(func() { released = true })
		assert.Equal(t, uint32(1), n.refs.Load())

		a := n.Anchor()

		require.NotZero(t, a, "should return non-nil client")
		require.False(t, released, "should not release before client")

		a.Release()
		assert.Eventually(t, func() bool {
			return released
		}, time.Second, time.Millisecond*10, "should release after client")
	})
}
