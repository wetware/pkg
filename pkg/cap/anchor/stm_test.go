package anchor_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wetware/ww/pkg/cap/anchor"
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

func TestGetOrCreate(t *testing.T) {
	t.Parallel()

	sched := anchor.NewScheduler(anchor.NewPath(""))

	for _, path := range []anchor.Path{
		anchor.NewPath("foo"),
		anchor.NewPath("foo/bar"),
		anchor.NewPath("foo/bar/baz"),
		anchor.NewPath("foo/bar/qux"),
	} {
		t.Run(trimmed(path.String()), func(t *testing.T) {
			tx := sched.Txn(true)
			defer tx.Finish()

			a, err := tx.GetOrCreate(path)
			require.NoError(t, err, "should insert %s", a)
			require.Equal(t, path.String(), a.Path().String())

			tx.Commit()

			// Try again with a read transaction.
			tx = sched.Txn(false)

			require.NotPanics(t, func() {
				a, err = tx.GetOrCreate(path)
			}, "should not panic - did it try to write?")

			require.NoError(t, err, "should insert %s", a)
			require.Equal(t, path.String(), a.Path().String())
		})
	}
}

func TestChildren(t *testing.T) {
	t.Parallel()

	var (
		sched    = anchor.NewScheduler(anchor.NewPath(""))
		children []anchor.Path
	)

	// /foo
	// /foo/bar
	// /foo/bar/baz
	// /foo/bar/qux
	insertTestValues(t, sched)

	tx := sched.Txn(false)
	cs, err := tx.Children()
	require.NoError(t, err)
	require.NotNil(t, cs)

	for v := cs.Next(); v != nil; v = cs.Next() {
		children = append(children, v.(anchor.Anchor).Path())
	}

	assert.Len(t, children, 1)
	assert.Contains(t, children, anchor.NewPath("foo"))
}

func TestWalkLongestSubpath(t *testing.T) {
	t.Parallel()
	t.Helper()

	var sched = anchor.NewScheduler(anchor.NewPath(""))

	// /foo
	// /foo/bar
	// /foo/bar/baz
	// /foo/bar/qux
	insertTestValues(t, sched)

	t.Run("ExactMatch", func(t *testing.T) {
		t.Parallel()

		tx := sched.Txn(false)

		a, err := tx.WalkLongestSubpath(anchor.NewPath("/foo/bar"))
		require.NoError(t, err, "should succeed")
		require.NotZero(t, a, "should return an anchor")
		require.Equal(t, "/foo/bar", a.Path().String())
	})

	t.Run("PartialMatch", func(t *testing.T) {
		t.Parallel()

		tx := sched.Txn(false)

		a, err := tx.WalkLongestSubpath(anchor.NewPath("/foo/bar/missing"))
		require.NoError(t, err, "should succeed")
		require.NotZero(t, a, "should return an anchor")
		require.Equal(t, "/foo/bar", a.Path().String())
	})

	t.Run("NoMatch", func(t *testing.T) {
		t.Parallel()

		tx := sched.Txn(false)

		a, err := tx.WalkLongestSubpath(anchor.NewPath("/missing"))
		require.NoError(t, err, "should succeed")
		require.Zero(t, a, "should return an empty anchor")
		require.True(t, a.Path().IsRoot(), "should be root anchor")
	})

	t.Run("Root", func(t *testing.T) {
		t.Parallel()

		// Root should fetch the node matching the scheduler's root path.
		tx := sched.
			WithSubpath(anchor.NewPath("/foo")).
			Txn(false)

		a, err := tx.WalkLongestSubpath(anchor.NewPath("/"))
		require.NoError(t, err, "should succeed")
		require.NotZero(t, a, "should return an anchor")
		require.Equal(t, "/foo", a.Path().String(),
			"anchor path should match scheduler's root path")
	})
}

func insertTestValues(t *testing.T, sched anchor.Scheduler) {
	tx := sched.Txn(true)
	defer tx.Finish()

	for _, path := range []anchor.Path{
		anchor.NewPath("foo"),
		anchor.NewPath("foo/bar"),
		anchor.NewPath("foo/bar/baz"),
		anchor.NewPath("foo/bar/qux"),
	} {
		a, err := tx.GetOrCreate(path)
		require.NoError(t, err, "should insert %s", a)
		require.Equal(t, path.String(), a.Path().String())
	}

	tx.Commit()
}
