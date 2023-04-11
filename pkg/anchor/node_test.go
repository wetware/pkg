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

	var wc weakClient
	assert.False(t, wc.Exists(), "zero-value weakClient should be null")
	assert.Panics(t, func() { wc.AddRef() }, "should panic on nil *capnp.WeakClient")

	client := capnp.ErrorClient(errors.New("test")) // non-null client
	wc.WeakClient = client.WeakRef()
	assert.True(t, wc.Exists(), "*capnp.WeakClient should exist")
	assert.True(t, client.IsSame(wc.AddRef()),
		"should return strong reference to underlying *WeakClient")
}

func TestNode_UseAfterFree(t *testing.T) {
	t.Parallel()

	var released bool
	n := mknode(nil, func() { released = true })

	n.Release()
	assert.True(t, released)

	assert.Panics(t, func() { n.AddRef() }, "should panic if AddRef() is called after free")
	assert.Panics(t, n.Release, "should panic if Release() is called after free")
}

func TestNode_Child(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("Create", func(t *testing.T) {
		t.Parallel()

		var released bool
		n := mknode(nil, func() { released = true })

		u := n.Child("child")
		require.NotEqual(t, n.nodestate, u.nodestate, "child should not be root anchor")

		u.Release()
		assert.True(t, released, "should release parent after releasing child")
	})

	t.Run("Exists", func(t *testing.T) {
		t.Parallel()

		var released bool
		n := mknode(nil, func() { released = true })

		u := n.Child("child")
		require.NotEqual(t, n.nodestate, u.nodestate,
			"child should not be root anchor")

		u2 := n.Child("child")
		require.NotEqual(t, n.nodestate, u2.nodestate,
			"child should not be root anchor")

		u.Release()
		require.False(t, released,
			"second child should keep root alive")

		u2.Release()
		assert.True(t, released,
			"should release parent after second child reference is released")
	})

	t.Run("Chain", func(t *testing.T) {
		t.Parallel()

		var released bool
		n := mknode(nil, func() { released = true })

		u := n.Child("foo").Child("bar").Child("baz")
		require.False(t, released, "should not release root")

		u.Release()
		assert.True(t, released, "should release full path")
	})
}

func TestNode_Anchor(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("Create", func(t *testing.T) {
		t.Parallel()

		var released atomic.Bool
		n := mknode(nil, func() { released.Store(true) })

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
		n := mknode(nil, func() { released.Store(true) })

		t.Log(n.refs.Load())

		a := n.Anchor()
		require.NotZero(t, a, "should return non-nil client")
		require.False(t, released.Load(),
			"should not release before client")

		t.Log(n.refs.Load())

		// NOTE: The first call to Anchor() has already captured n's
		// reference, so we must create a new one.
		b := n.AddRef().Anchor()
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
