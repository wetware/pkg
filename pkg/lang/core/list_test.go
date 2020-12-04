package core_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/core"
	capnp "zombiezen.com/go/capnproto2"
)

func TestEmptyList(t *testing.T) {
	t.Run("Count", func(t *testing.T) {
		cnt, err := core.EmptyList.Count()
		assert.NoError(t, err)
		assert.Zero(t, cnt)
	})

	t.Run("First", func(t *testing.T) {
		v, err := core.EmptyList.First()
		assert.NoError(t, err)
		assert.Nil(t, v)
	})

	t.Run("Next", func(t *testing.T) {
		tail, err := core.EmptyList.Next()
		assert.NoError(t, err)
		assert.Nil(t, tail)
	})

	t.Run("Pop", func(t *testing.T) {
		tail, err := core.EmptyList.Pop()
		assert.True(t, errors.Is(err, core.ErrIllegalState),
			"'%s' is not ErrIllegalState", err)

		assert.Nil(t, tail)
	})

	t.Run("Conj", func(t *testing.T) {
		seq, err := core.EmptyList.Conj(core.True)
		assert.NoError(t, err)

		cnt, err := seq.Count()
		assert.NoError(t, err)
		assert.Equal(t, 1, cnt)

		v, err := seq.First()
		assert.NoError(t, err)

		eq, err := core.Eq(core.True, v.(ww.Any))
		require.NoError(t, err)
		assert.True(t, eq)
	})
}

func TestListEquality(t *testing.T) {
	for _, tt := range []struct {
		desc    string
		l       core.List
		newList func() core.List
	}{
		{
			desc: "basic",
			l:    mustList(mustInt(0), mustInt(1), mustInt(2)),
			newList: func() core.List {
				return mustList(mustInt(0), mustInt(1), mustInt(2))
			},
		},
		{
			desc: "shared tail",
			l:    mustList(mustInt(0), mustInt(1), mustInt(2), mustInt(3)),
			newList: func() core.List {
				l := mustList(mustInt(1), mustInt(2), mustInt(3))
				l, err := l.Cons(mustInt(0))
				require.NoError(t, err)
				return l
			},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			eq, err := core.Eq(tt.l, tt.newList())
			require.NoError(t, err)

			assert.True(t, eq)
		})
	}
}

func mustList(vs ...ww.Any) core.List {
	l, err := core.NewList(capnp.SingleSegment(nil), vs...)
	if err != nil {
		panic(err)
	}

	return l
}
