package anchor

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"capnproto.org/go/capnp/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	api "github.com/wetware/ww/internal/api/anchor"
)

func TestWeakClient(t *testing.T) {
	t.Parallel()

	var wc weakClient
	assert.False(t, wc.Exists(), "zero-value weakClient should be null")
	assert.Panics(t, func() { wc.AddRef() }, "should panic on nil *capnp.WeakClient")

	var released bool
	client := capnp.ErrorClient(errors.New("test")) // non-null client
	wc.WeakClient = client.WeakRef()
	wc.releaser = mknode(func() { released = true })
	assert.True(t, wc.Exists(), "*capnp.WeakClient should exist")
	assert.True(t, client.IsSame(wc.AddRef()),
		"should return strong reference to underlying *WeakClient")

	wc.Release()
	assert.True(t, released, "should release refcounter")
	assert.False(t, wc.Exists(), "client should not exist after reset")
}

func TestWalk(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("BaseCase", func(t *testing.T) {
		t.Parallel()

		var released atomic.Bool
		n := mknode(func() { released.Store(true) })

		root := n.Anchor()
		defer root.Release()

		f, release := root.Walk(context.Background(), func(ps api.Anchor_walk_Params) error {
			return ps.SetPath("/")
		})
		defer release()

		res, err := f.Struct()
		require.NoError(t, err, "rpc should succeed")

		anchor := res.Anchor()
		defer anchor.Release()
		require.True(t, anchor.IsValid(), "client should not have been released")
		assert.True(t, anchor.IsSame(root), "should be root anchor")

		release()
		assert.False(t, released.Load(), "should not release node")
	})

	t.Run("N+1Case", func(t *testing.T) {
		t.Parallel()

		var released atomic.Bool
		n := mknode(func() { released.Store(true) })

		root := n.Anchor()
		defer root.Release()

		f, release := root.Walk(context.Background(), func(ps api.Anchor_walk_Params) error {
			return ps.SetPath("/child")
		})
		defer release()

		res, err := f.Struct()
		require.NoError(t, err, "rpc should succeed")

		anchor := res.Anchor()
		defer anchor.Release()
		require.True(t, anchor.IsValid(), "client should not have been released")
		assert.False(t, anchor.IsSame(root), "should not be root anchor")

		release()
		assert.False(t, released.Load(), "should not release node")
	})
}
