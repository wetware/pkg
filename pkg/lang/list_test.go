package lang_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/pkg/lang"
)

func TestEmptyList(t *testing.T) {
	t.Run("Count", func(t *testing.T) {
		cnt, err := lang.EmptyList.Count()
		assert.NoError(t, err)
		assert.Zero(t, cnt)
	})

	t.Run("First", func(t *testing.T) {
		v, err := lang.EmptyList.First()
		assert.NoError(t, err)
		assert.Nil(t, v)
	})

	t.Run("Next", func(t *testing.T) {
		tail, err := lang.EmptyList.Next()
		assert.NoError(t, err)
		assert.Nil(t, tail)
	})

	t.Run("Conj", func(t *testing.T) {
		seq, err := lang.EmptyList.Conj(lang.True)
		assert.NoError(t, err)

		cnt, err := seq.Count()
		assert.NoError(t, err)
		assert.Equal(t, 1, cnt)

		v, err := seq.First()
		assert.NoError(t, err)
		assert.Equal(t, mustSExpr(lang.True), mustSexpr(v))
	})
}

func TestListEquality(t *testing.T) {
	for _, tt := range []struct {
		desc    string
		l       lang.List
		newList func() lang.List
	}{
		{
			desc: "basic",
			l:    mustList(mustInt(0), mustInt(1), mustInt(2)),
			newList: func() lang.List {
				return mustList(mustInt(0), mustInt(1), mustInt(2))
			},
		},
		{
			desc: "shared tail",
			l:    mustList(mustInt(0), mustInt(1), mustInt(2), mustInt(3)),
			newList: func() lang.List {
				l := mustList(mustInt(1), mustInt(2), mustInt(3))
				l, err := l.Cons(mustInt(0))
				require.NoError(t, err)
				return l
			},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			assert.True(t, lang.Eq(tt.l, tt.newList()),
				"expected %s, got %s", mustSexpr(tt.l), mustSexpr(tt.newList()))
		})
	}
}
