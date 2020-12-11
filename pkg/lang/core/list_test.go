package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/core"
	capnp "zombiezen.com/go/capnproto2"
)

func TestEmptyList(t *testing.T) {
	t.Parallel()

	t.Run("New", func(t *testing.T) {
		t.Parallel()

		l, err := core.NewList(capnp.SingleSegment(nil))
		require.NoError(t, err)
		assert.IsType(t, core.EmptyPersistentList{}, l)
	})

	t.Run("Count", func(t *testing.T) {
		t.Parallel()

		cnt, err := core.EmptyList.Count()
		assert.NoError(t, err)
		assert.Zero(t, cnt)
	})

	t.Run("Render", func(t *testing.T) {
		t.Parallel()

		s, err := core.Render(core.EmptyList)
		require.NoError(t, err)
		assert.Equal(t, "()", s)
	})

	t.Run("First", func(t *testing.T) {
		t.Parallel()

		item, err := core.EmptyList.First()
		assert.NoError(t, err)
		assert.Nil(t, item)
	})

	t.Run("Next", func(t *testing.T) {
		t.Parallel()

		tail, err := core.EmptyList.Next()
		assert.NoError(t, err)
		assert.Nil(t, tail)
	})

	t.Run("Cons", func(t *testing.T) {
		t.Parallel()

		l, err := core.EmptyList.Cons(mustInt(0))
		require.NoError(t, err)
		require.IsType(t, core.PersistentHeadList{}, l)

		cnt, err := l.Count()
		require.NoError(t, err)
		assert.Equal(t, 1, cnt)
	})

	t.Run("Conj", func(t *testing.T) {
		t.Parallel()

		t.Run("One", func(t *testing.T) {
			t.Parallel()

			ctr, err := core.EmptyList.Conj(valueRange(1)...)
			require.NoError(t, err)
			assert.IsType(t, core.PersistentHeadList{}, ctr)
		})

		t.Run("Two", func(t *testing.T) {
			t.Parallel()

			ctr, err := core.EmptyList.Conj(valueRange(2)...)
			require.NoError(t, err)
			assert.IsType(t, core.PackedPersistentList{}, ctr)
		})

		t.Run("Many", func(t *testing.T) {
			t.Parallel()

			ctr, err := core.EmptyList.Conj(valueRange(3)...)
			require.NoError(t, err)
			assert.IsType(t, core.DeepPersistentList{}, ctr)
		})
	})

	t.Run("Iter", func(t *testing.T) {
		t.Parallel()

		s, err := core.ToSlice(core.EmptyList)
		require.NoError(t, err)
		assert.Nil(t, s)
	})
}

func TestPersistentHeadList(t *testing.T) {
	t.Parallel()

	t.Run("New", func(t *testing.T) {
		t.Parallel()

		l, err := core.NewList(capnp.SingleSegment(nil), mustInt(0))
		require.NoError(t, err)
		assert.IsType(t, core.PersistentHeadList{}, l)
	})

	t.Run("Count", func(t *testing.T) {
		t.Parallel()

		l, err := core.NewList(capnp.SingleSegment(nil), mustInt(0))
		require.NoError(t, err)

		cnt, err := l.Count()
		require.NoError(t, err)
		assert.Equal(t, 1, cnt)
	})

	t.Run("Render", func(t *testing.T) {
		t.Parallel()

		l, err := core.NewList(capnp.SingleSegment(nil), mustInt(0))
		require.NoError(t, err)

		s, err := core.Render(l)
		require.NoError(t, err)
		assert.Equal(t, "(0)", s)
	})

	t.Run("First", func(t *testing.T) {
		t.Parallel()

		l, err := core.NewList(capnp.SingleSegment(nil), mustInt(0))
		require.NoError(t, err)

		item, err := l.First()
		require.NoError(t, err)
		assertEq(t, mustInt(0), item)
	})

	t.Run("Next", func(t *testing.T) {
		t.Parallel()

		l, err := core.NewList(capnp.SingleSegment(nil), mustInt(0))
		require.NoError(t, err)

		seq, err := l.Next()
		require.NoError(t, err)
		assert.Nil(t, seq)
	})

	t.Run("Cons", func(t *testing.T) {
		t.Parallel()

		l, err := core.NewList(capnp.SingleSegment(nil), mustInt(0))
		require.NoError(t, err)

		l, err = l.Cons(mustInt(1))
		require.NoError(t, err)
		assert.IsType(t, core.PackedPersistentList{}, l)

		got, err := l.First()
		require.NoError(t, err)
		assertEq(t, mustInt(1), got)
	})

	t.Run("Conj", func(t *testing.T) {
		t.Parallel()

		t.Run("One", func(t *testing.T) {
			l, err := core.NewList(capnp.SingleSegment(nil), mustInt(0))
			require.NoError(t, err)
			require.IsType(t, core.PersistentHeadList{}, l)

			ctr, err := l.Conj(valueRange(1)...)
			require.NoError(t, err)
			require.IsType(t, core.PackedPersistentList{}, ctr)

			got, err := l.First()
			require.NoError(t, err)
			assertEq(t, mustInt(0), got)
		})

		t.Run("Many", func(t *testing.T) {
			l, err := core.NewList(capnp.SingleSegment(nil), mustInt(0))
			require.NoError(t, err)
			require.IsType(t, core.PersistentHeadList{}, l)

			ctr, err := l.Conj(core.True, core.False)
			require.NoError(t, err)
			require.IsType(t, core.DeepPersistentList{}, ctr)

			got, err := l.First()
			require.NoError(t, err)
			assertEq(t, core.False, got)
		})
	})

	t.Run("Iter", func(t *testing.T) {
		l, err := core.NewList(capnp.SingleSegment(nil), mustInt(0))
		require.NoError(t, err)

		s, err := core.ToSlice(l)
		require.NoError(t, err)
		assert.Len(t, s, 1)

		assertEq(t, s[0], mustInt(0))
	})
}

func TestPackedPersistentList(t *testing.T) {
	t.Parallel()

	items := valueRange(2)

	t.Run("New", func(t *testing.T) {
		t.Parallel()

		l, err := core.NewList(capnp.SingleSegment(nil), items...)
		require.NoError(t, err)
		assert.IsType(t, core.PackedPersistentList{}, l)
	})

	t.Run("Count", func(t *testing.T) {
		t.Parallel()

		l, err := core.NewList(capnp.SingleSegment(nil), items...)
		require.NoError(t, err)

		cnt, err := l.Count()
		require.NoError(t, err)
		assert.Equal(t, 2, cnt)
	})

	t.Run("Render", func(t *testing.T) {
		t.Parallel()

		l, err := core.NewList(capnp.SingleSegment(nil), items...)
		require.NoError(t, err)

		s, err := core.Render(l)
		require.NoError(t, err)
		assert.Equal(t, "(0 1)", s)
	})

	t.Run("First", func(t *testing.T) {
		t.Parallel()

		l, err := core.NewList(capnp.SingleSegment(nil), items...)
		require.NoError(t, err)

		item, err := l.First()
		require.NoError(t, err)

		eq, err := core.Eq(items[0], item)
		require.NoError(t, err)
		assert.True(t, eq)
	})

	t.Run("Next", func(t *testing.T) {
		t.Parallel()

		l, err := core.NewList(capnp.SingleSegment(nil), items...)
		require.NoError(t, err)

		seq, err := l.Next()
		require.NoError(t, err)
		assert.IsType(t, core.PersistentHeadList{}, seq)
	})

	t.Run("Cons", func(t *testing.T) {
		t.Parallel()

		l, err := core.NewList(capnp.SingleSegment(nil), items...)
		require.NoError(t, err)

		l, err = l.Cons(mustInt(2))
		require.NoError(t, err)
		assert.IsType(t, core.DeepPersistentList{}, l)

		got, err := l.First()
		require.NoError(t, err)
		assertEq(t, mustInt(2), got)
	})

	t.Run("Conj", func(t *testing.T) {
		t.Parallel()

		t.Run("One", func(t *testing.T) {
			t.Parallel()

			l, err := core.NewList(capnp.SingleSegment(nil), items...)
			require.NoError(t, err)
			require.IsType(t, core.PackedPersistentList{}, l)

			ctr, err := l.Conj(core.True)
			require.NoError(t, err)
			require.IsType(t, core.DeepPersistentList{}, ctr)

			got, err := l.First()
			require.NoError(t, err)
			assertEq(t, core.True, got)
		})

		t.Run("Many", func(t *testing.T) {
			t.Parallel()

			l, err := core.NewList(capnp.SingleSegment(nil), items...)
			require.NoError(t, err)
			require.IsType(t, core.PackedPersistentList{}, l)

			ctr, err := l.Conj(core.True, core.False)
			require.NoError(t, err)
			require.IsType(t, core.DeepPersistentList{}, ctr)

			got, err := l.First()
			require.NoError(t, err)
			assertEq(t, core.False, got)
		})
	})

	t.Run("Iter", func(t *testing.T) {
		l, err := core.NewList(capnp.SingleSegment(nil), items...)
		require.NoError(t, err)

		var i int
		require.NoError(t, core.ForEach(l, func(item ww.Any) (bool, error) {
			assertEq(t, items[i], item)
			i++
			return false, nil
		}))
	})
}

func TestDeepPersistentList(t *testing.T) {
	t.Parallel()

	items := valueRange(3)

	t.Run("New", func(t *testing.T) {
		t.Parallel()

		l, err := core.NewList(capnp.SingleSegment(nil), items...)
		require.NoError(t, err)
		assert.IsType(t, core.DeepPersistentList{}, l)
	})

	t.Run("Count", func(t *testing.T) {
		t.Parallel()

		l, err := core.NewList(capnp.SingleSegment(nil), items...)
		require.NoError(t, err)

		cnt, err := l.Count()
		require.NoError(t, err)
		assert.Equal(t, len(items), cnt)
	})

	t.Run("Render", func(t *testing.T) {
		t.Parallel()

		l, err := core.NewList(capnp.SingleSegment(nil), items...)
		require.NoError(t, err)

		s, err := core.Render(l)
		require.NoError(t, err)
		assert.Equal(t, "(0 1 2)", s)
	})

	t.Run("First", func(t *testing.T) {
		t.Parallel()

		l, err := core.NewList(capnp.SingleSegment(nil), items...)
		require.NoError(t, err)

		item, err := l.First()
		require.NoError(t, err)

		assertEq(t, items[0], item)
	})

	t.Run("Next", func(t *testing.T) {
		t.Parallel()

		l, err := core.NewList(capnp.SingleSegment(nil), items...)
		require.NoError(t, err)

		seq, err := l.Next()
		require.NoError(t, err)
		assert.IsType(t, core.PackedPersistentList{}, seq)
	})

	t.Run("Cons", func(t *testing.T) {
		t.Parallel()

		l, err := core.NewList(capnp.SingleSegment(nil), items...)
		require.NoError(t, err)
		require.IsType(t, core.DeepPersistentList{}, l)

		l, err = l.Cons(mustInt(3))
		require.NoError(t, err)
		assert.IsType(t, core.DeepPersistentList{}, l)

		cnt, err := l.Count()
		require.NoError(t, err)
		assert.Equal(t, len(items)+1, cnt)

		got, err := l.First()
		require.NoError(t, err)
		assertEq(t, mustInt(3), got)
	})

	t.Run("Conj", func(t *testing.T) {
		t.Parallel()

		l, err := core.NewList(capnp.SingleSegment(nil), items...)
		require.NoError(t, err)

		ctr, err := l.Conj(core.True, core.False)
		require.NoError(t, err)
		require.IsType(t, core.DeepPersistentList{}, ctr)

		cnt, err := ctr.Count()
		require.NoError(t, err)
		assert.Equal(t, len(items)+2, cnt)

		got, err := l.First()
		require.NoError(t, err)
		assertEq(t, core.False, got)
	})

	t.Run("Iter", func(t *testing.T) {
		l, err := core.NewList(capnp.SingleSegment(nil), items...)
		require.NoError(t, err)

		var i int
		require.NoError(t, core.ForEach(l, func(item ww.Any) (bool, error) {
			assertEq(t, items[i], item)
			i++
			return false, nil
		}))
	})
}

func mustList(vs ...ww.Any) core.List {
	l, err := core.NewList(capnp.SingleSegment(nil), vs...)
	if err != nil {
		panic(err)
	}

	return l
}

func assertEq(t *testing.T, want, got ww.Any) {
	eq, err := core.Eq(want, got)
	require.NoError(t, err, "core.Eq returned an error")
	assert.True(t, eq, "mem values are not equal")
}
