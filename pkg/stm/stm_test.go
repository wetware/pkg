package stm_test

import (
	"testing"

	"github.com/hashicorp/go-memdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/pkg/stm"
)

func TestSTM(t *testing.T) {
	t.Parallel()

	var (
		f       stm.Factory
		s, snap stm.Scheduler
		ref     stm.TableRef
	)

	t.Run("Factory", func(t *testing.T) {
		ref = f.Register("test", &memdb.TableSchema{
			Name: "test",
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &memdb.BoolFieldIndex{Field: "Test"},
				},
			},
		})
		assert.NotZero(t, ref, "table ref should be populated")

		assert.Panics(t, func() { _ = f.Register("test", nil) },
			"duplicate table name should cause panic")

		var err error
		s, err = f.NewScheduler()
		require.NoError(t, err, "schema should be valid")
		assert.NotZero(t, s, "should allocate scheduler")

		// We take a snapshot of the initial (empty) state of the
		// scheduler. After committing our writes, we will verify
		// that it has not mutated.
		snap = s.Snapshot()
	})

	t.Run("Insert", func(t *testing.T) {
		tx := s.Txn(true)
		defer tx.Commit()

		err := tx.Insert(ref, struct{ Test bool }{Test: true})
		assert.NoError(t, err, "should insert 'true' case")

		err = tx.Insert(ref, struct{ Test bool }{})
		assert.NoError(t, err, "should insert 'false' case")
	})

	t.Run("Query", func(t *testing.T) {
		t.Helper()

		t.Run("Get", func(t *testing.T) {
			t.Parallel()

			tx := s.Txn(false)
			defer tx.Commit()

			it, err := tx.Get(ref, "id", true)
			require.NoError(t, err, "query 'get' should succeed")

			var results []bool
			for v := it.Next(); v != nil; v = it.Next() {
				t.Logf("got:\t%v", v)
				results = append(results, v.(struct{ Test bool }).Test)
			}

			assert.Contains(t, results, true)
			assert.NotContains(t, results, false)
		})
	})

	t.Run("Snapshot", func(t *testing.T) {
		t.Parallel()

		// Now let's check that the snaphsot was not mutated.
		tx := snap.Txn(false)

		it, err := tx.Get(ref, "id", true)
		require.NoError(t, err, "query 'get' should succeed")
		assert.Nil(t, it.Next(), "iterator should be exhausted")

		it, err = tx.Get(ref, "id", false)
		require.NoError(t, err, "query 'get' should succeed")
		assert.Nil(t, it.Next(), "iterator should be exhausted")
	})
}
