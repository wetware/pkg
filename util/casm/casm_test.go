package casm_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wetware/pkg/util/casm"
)

func TestIterator(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("Zero", func(t *testing.T) {
		t.Parallel()

		/*
			Test that the zero-value Iterator is empty.
		*/

		it := casm.Iterator[string]{}

		s, ok := it.Next()
		assert.Zero(t, s, "should return zero-value string")
		assert.False(t, ok, "should be exhausted")

		assert.NoError(t, it.Err(), "should not encounter error")
	})

	t.Run("Succeed", func(t *testing.T) {
		t.Parallel()

		seq := mockSeq{"hello, world!"}

		it := casm.Iterator[string]{
			Future: context.Background(),
			Seq:    &seq,
		}

		got, ok := it.Next()
		assert.Equal(t, "hello, world!", got, "should return sequence value")
		assert.True(t, ok, "should not be exhausted")

		got, ok = it.Next()
		assert.Zero(t, got, "should return zero-value string")
		assert.False(t, ok, "should be exhausted")

		assert.NoError(t, it.Err(), "should have succeeded")
	})

	t.Run("Abort", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		seq := mockSeq{"hello, world!"}

		it := casm.Iterator[string]{
			Future: ctx,
			Seq:    &seq,
		}

		/*
			NOTE:	the iterator MUST NOT abort until buffered items
					have been consumed!
		*/

		got, ok := it.Next()
		assert.Equal(t, "hello, world!", got, "should return sequence value")
		assert.True(t, ok,
			"should consume buffered items before aborting")

		got, ok = it.Next()
		assert.Zero(t, got, "should return zero-value string")
		assert.False(t, ok, "should be exhausted")

		assert.ErrorIs(t, it.Err(), context.Canceled,
			"should abort with context.Canceled")
	})
}

type mockSeq []string

func (seq *mockSeq) Next() (head string, ok bool) {
	if ok = len(*seq) > 0; ok {
		head, *seq = (*seq)[0], (*seq)[1:]
		ok = true
	}

	return
}
