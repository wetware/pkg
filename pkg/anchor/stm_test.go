package anchor_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/pkg/anchor"
)

func TestNewScheduler(t *testing.T) {
	t.Parallel()

	// Ensure constructor does not panic due to table name collision.

	var sched anchor.Scheduler
	require.NotPanics(t, func() {
		sched = anchor.NewScheduler(anchor.NewPath(""))
	}, "table name collision?")

	require.NotZero(t, sched, "root anchor should be initialized")
}

// func TestGetOrCreate(t *testing.T) {
// 	t.Parallel()

// 	sched := anchor.NewScheduler(anchor.NewPath(""))

// 	for _, path := range []anchor.Path{
// 		anchor.NewPath("foo"),
// 		anchor.NewPath("foo/bar"),
// 		anchor.NewPath("foo/bar/baz"),
// 		anchor.NewPath("foo/bar/qux"),
// 	} {
// 		t.Run(trimmed(path.String()), func(t *testing.T) {
// 			tx := sched.Txn(true)
// 			defer tx.Finish()

// 			a, err := tx.GetOrCreate(path)
// 			require.NoError(t, err, "should insert %s", a)
// 			require.Equal(t, path.String(), a.Path().String())

// 			tx.Commit()

// 			// Try again with a read transaction.
// 			tx = sched.Txn(false)

// 			require.NotPanics(t, func() {
// 				a, err = tx.GetOrCreate(path)
// 			}, "should not panic - did it try to write?")

// 			require.NoError(t, err, "should insert %s", a)
// 			require.Equal(t, path.String(), a.Path().String())
// 		})
// 	}
// }

// func TestChildren(t *testing.T) {
// 	t.Parallel()
// 	t.Helper()

// 	sched := anchor.NewScheduler(anchor.NewPath(""))

// 	// /foo
// 	// /foo/bar
// 	// /foo/bar/baz
// 	// /foo/bar/qux
// 	insertTestValues(t, sched)

// 	t.Run("Root", func(t *testing.T) {
// 		t.Parallel()

// 		tx := sched.Txn(false)

// 		cs, err := tx.Children()
// 		require.NoError(t, err)
// 		require.NotNil(t, cs)

// 		var children []anchor.Path
// 		for v := cs.Next(); v != nil; v = cs.Next() {
// 			children = append(children, v.(anchor.Server).Path())
// 		}

// 		// Should have single child:  /foo
// 		assert.Len(t, children, 1, "should have exactly one child")
// 		assert.Contains(t, children, anchor.NewPath("/foo"),
// 			"child should be /foo")
// 	})

// 	t.Run("foo", func(t *testing.T) {
// 		t.Parallel()

// 		tx := sched.
// 			WithSubpath(anchor.NewPath("foo")).
// 			Txn(false)

// 		cs, err := tx.Children()
// 		require.NoError(t, err)
// 		require.NotNil(t, cs)

// 		var children []anchor.Path
// 		for v := cs.Next(); v != nil; v = cs.Next() {
// 			children = append(children, v.(anchor.Server).Path())
// 		}

// 		// Should have single child
// 		assert.Len(t, children, 1, "should have exactly one child")
// 		assert.Contains(t, children, anchor.NewPath("/foo/bar"),
// 			"child should be /foo/bar")
// 	})

// 	t.Run("foo/bar", func(t *testing.T) {
// 		t.Parallel()

// 		tx := sched.
// 			WithSubpath(anchor.NewPath("/foo/bar")).
// 			Txn(false)

// 		cs, err := tx.Children()
// 		require.NoError(t, err)
// 		require.NotNil(t, cs)

// 		var children []anchor.Path
// 		for v := cs.Next(); v != nil; v = cs.Next() {
// 			children = append(children, v.(anchor.Server).Path())
// 		}

// 		// Should have two children
// 		assert.Len(t, children, 2,
// 			"should have exactly two children")
// 		assert.Contains(t, children, anchor.NewPath("/foo/bar/baz"),
// 			"should have child /foo/bar/baz")
// 		assert.Contains(t, children, anchor.NewPath("/foo/bar/qux"),
// 			"should have child /foo/bar/qux")
// 	})
// }

// func TestWalkLongestSubpath(t *testing.T) {
// 	t.Parallel()
// 	t.Helper()

// 	var sched = anchor.NewScheduler(anchor.NewPath(""))

// 	// /foo
// 	// /foo/bar
// 	// /foo/bar/baz
// 	// /foo/bar/qux
// 	insertTestValues(t, sched)

// 	t.Run("ExactMatch", func(t *testing.T) {
// 		t.Parallel()

// 		tx := sched.Txn(false)

// 		a, err := tx.WalkLongestSubpath(anchor.NewPath("/foo/bar"))
// 		require.NoError(t, err, "should succeed")
// 		require.NotZero(t, a, "should return an anchor")
// 		require.Equal(t, "/foo/bar", a.Path().String())
// 	})

// 	t.Run("PartialMatch", func(t *testing.T) {
// 		t.Parallel()

// 		tx := sched.Txn(false)

// 		a, err := tx.WalkLongestSubpath(anchor.NewPath("/foo/bar/missing"))
// 		require.NoError(t, err, "should succeed")
// 		require.NotZero(t, a, "should return an anchor")
// 		require.Equal(t, "/foo/bar", a.Path().String())
// 	})

// 	t.Run("NoMatch", func(t *testing.T) {
// 		t.Parallel()

// 		tx := sched.Txn(false)

// 		a, err := tx.WalkLongestSubpath(anchor.NewPath("/missing"))
// 		require.NoError(t, err, "should succeed")
// 		require.Zero(t, a, "should return an empty anchor")
// 		require.True(t, a.Path().IsRoot(), "should be root anchor")
// 	})

// 	t.Run("Root", func(t *testing.T) {
// 		t.Parallel()

// 		// Root should fetch the node matching the scheduler's root path.
// 		tx := sched.
// 			WithSubpath(anchor.NewPath("/foo")).
// 			Txn(false)

// 		a, err := tx.WalkLongestSubpath(anchor.NewPath("/"))
// 		require.NoError(t, err, "should succeed")
// 		require.NotZero(t, a, "should return an anchor")
// 		require.Equal(t, "/foo", a.Path().String(),
// 			"anchor path should match scheduler's root path")
// 	})
// }

// func insertTestValues(t *testing.T, sched anchor.Scheduler) {
// 	tx := sched.Txn(true)
// 	defer tx.Finish()

// 	for _, path := range []anchor.Path{
// 		anchor.NewPath("foo"),
// 		anchor.NewPath("foo/bar"),
// 		anchor.NewPath("foo/bar/baz"),
// 		anchor.NewPath("foo/bar/qux"),
// 	} {
// 		a, err := tx.GetOrCreate(path)
// 		require.NoError(t, err, "should insert %s", a)
// 		require.Equal(t, path.String(), a.Path().String())
// 	}

// 	tx.Commit()
// }
