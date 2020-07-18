package host

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestList(t *testing.T) {
	root := newAnchorTree()

	assert.Empty(t, root.List())
	defer assert.Empty(t, root.List())

	test := root.Walk([]string{"test"})
	t.Log(test.Path(), test.List())

	assert.Empty(t, test.List())
	assert.NotEmpty(t, root.List())
}

func TestWalk(t *testing.T) {
	t.Run("test", func(t *testing.T) {
		root := newAnchorTree()
		assert.Empty(t, root.List())
		defer assert.Empty(t, root.List())

		t.Log(root.Path(), root.List())
		assert.Equal(t, []string{}, root.Path())

		test := root.Walk([]string{"test"})
		t.Log(test.Path(), test.List())

		assert.Empty(t, test.List())
		assert.NotEmpty(t, root.List())

	})

	t.Run("test/foo/bar/baz", func(t *testing.T) {
		root := newAnchorTree()
		assert.Empty(t, root.List())
		defer assert.Empty(t, root.List())

		t.Log(root.Path(), root.List())
		assert.Equal(t, []string{}, root.Path())

		test := root.Walk([]string{"test", "foo", "bar", "baz"})
		t.Log(test.Path(), test.List())

		assert.Empty(t, test.List())
		assert.NotEmpty(t, root.List())
	})

	t.Run("test/foo;test/bar;test/baz", func(t *testing.T) {
		root := newAnchorTree()
		assert.Empty(t, root.List())
		defer assert.Empty(t, root.List())

		t.Log(root.Path(), root.List())
		assert.Equal(t, []string{}, root.Path())

		table := [][]string{
			{"alpha", "one"},
			{"alpha", "two"},
			{"alpha", "three"},
			{"alpha", "four"},
			{"bravo", "one"},
			{"bravo", "two"},
			{"bravo", "three"},
			{"bravo", "four"},
		}

		var wg sync.WaitGroup
		wg.Add(len(table))

		for _, path := range table {
			go func(path []string) {
				defer wg.Done()

				// ["test", *]
				subtest := root.Walk(path)

				t.Log(subtest.Path(), subtest.List())

				assert.Empty(t, subtest.List()) // leaves should have no children
				assert.NotEmpty(t, root.List()) // root should be non-empty
			}(path)
		}

		wg.Wait()
	})
}

func discard(anchorNode) {}

func BenchmarkWalk(b *testing.B) {
	root := newAnchorTree()

	b.Run("SimpleRepeatInsert", func(b *testing.B) {
		key := []string{"test"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			discard(root.Walk(key))
		}
	})

	b.Run("ComplexInsert", func(b *testing.B) {
		table := [][]string{
			{"alpha", "one"},
			{"alpha", "two"},
			{"alpha", "three"},
			{"alpha", "four"},
			{"bravo", "one"},
			{"bravo", "two"},
			{"bravo", "three"},
			{"bravo", "four"},
		}

		var wg sync.WaitGroup

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			wg.Add(len(table))

			for _, path := range table {
				go func(path []string) {
					defer wg.Done()

					discard(root.Walk(path))
				}(path)
			}
		}

		wg.Wait()
	})
}
