package lang_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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
		assert.Equal(t, lang.True.String(), v.(fmt.Stringer).String())
	})
}
