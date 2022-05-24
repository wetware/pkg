package stm_test

// import (
// 	"testing"

// 	"github.com/wetware/ww/pkg/cap/cluster"
// 	"github.com/wetware/ww/pkg/stm"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// )

// func TestPath(t *testing.T) {
// 	t.Parallel()

// 	for _, tt := range []struct {
// 		name, want string
// 		got        cluster.Path
// 		parts      [][]byte
// 	}{
// 		{
// 			got:  cluster.NewPath(""),
// 			want: "/",
// 		},
// 		{
// 			name:  "foo",
// 			got:   cluster.NewPath("foo"),
// 			want:  "foo/",
// 			parts: [][]byte{[]byte("foo/")},
// 		},
// 		{
// 			name:  "bar",
// 			got:   cluster.NewPath("foo/bar"),
// 			want:  "foo/bar/",
// 			parts: [][]byte{[]byte("foo/"), []byte("bar/")},
// 		},
// 		{
// 			name:  "baz",
// 			got:   cluster.NewPath("foo/bar/baz"),
// 			want:  "foo/bar/baz/",
// 			parts: [][]byte{[]byte("foo/"), []byte("bar/"), []byte("baz/")},
// 		},
// 		{
// 			name:  "qux",
// 			got:   cluster.NewPath("foo/bar/qux"),
// 			want:  "foo/bar/qux/",
// 			parts: [][]byte{[]byte("foo/"), []byte("bar/"), []byte("qux/")},
// 		},
// 	} {
// 		assert.Equal(t, []byte(tt.want), tt.got.Bytes(),
// 			"should have terminating '/' byte")
// 		assert.Equal(t, tt.want[:len(tt.want)-1], tt.got.String(),
// 			"string values should match")
// 		assert.Equal(t, tt.name, tt.got.Name(),
// 			"names should match")
// 		assert.Equal(t, tt.parts, tt.got.Parts(),
// 			"parts should match and have terminating '/' byte")
// 	}
// }

// func TestSTM(t *testing.T) {
// 	t.Parallel()
// 	t.Helper()

// 	var (
// 		f   stm.Factory
// 		s   stm.Scheduler
// 		ref stm.TableRef
// 	)

// 	t.Run("Init", func(t *testing.T) {
// 		require.NotPanics(t, func() {
// 			ref = f.Register("anchor", &cluster.AnchorSchema)
// 		}, "table 'anchor' already exists")

// 		var err error
// 		s, err = f.NewScheduler()
// 		require.NoError(t, err, "should create scheduler")
// 	})

// 	t.Run("Insert", func(t *testing.T) {
// 		tx := s.Txn(true)
// 		defer tx.Commit()

// 		for _, a := range []*cluster.Anchor{
// 			anchor("foo"),
// 			anchor("foo/bar"),
// 			anchor("foo/bar/baz"),
// 			anchor("foo/bar/qux"),
// 		} {
// 			err := tx.Insert(ref, a)
// 			assert.NoError(t, err, "should insert %s", a)
// 		}
// 	})

// 	t.Run("QueryAnchor", func(t *testing.T) {
// 		tx := s.Txn(false) // read-only tx
// 		defer tx.Commit()

// 		for _, p := range []cluster.Path{
// 			cluster.NewPath("foo"),
// 			cluster.NewPath("foo/bar"),
// 			cluster.NewPath("foo/bar/baz"),
// 			cluster.NewPath("foo/bar/qux"),
// 		} {
// 			v, err := tx.First(ref, "id", p)
// 			require.NoError(t, err,
// 				"query should succeed")
// 			require.IsType(t, new(cluster.Anchor), v,
// 				"query should return an anchor")
// 			require.Equal(t, p.String(), v.(*cluster.Anchor).Path.String(),
// 				"returned anchor's path should match")
// 		}
// 	})

// 	// t.Run("QueryXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX", func(t *testing.T) {
// 	// 	tx := s.Txn(false) // read-only tx
// 	// 	defer tx.Commit()

// 	// 	for _, p := range []cluster.Path{
// 	// 		cluster.NewPath("foo"),
// 	// 		cluster.NewPath("foo/bar"),
// 	// 		cluster.NewPath("foo/bar/baz"),
// 	// 		cluster.NewPath("foo/bar/qux"),
// 	// 	} {
// 	// 		it, err := tx.Get(ref, "id", p)
// 	// 		require.NoError(t, err, "query should succeed")

// 	// 		var results []*cluster.Anchor
// 	// 		for v := it.Next(); v != nil; v = it.Next() {
// 	// 			results = append(results, v.(*cluster.Anchor))
// 	// 		}

// 	// 		t.Log(results)
// 	// 		t.FailNow()
// 	// 		require.NotEmpty(t, results, "result set shoudl not be empty")
// 	// 	}
// 	// })

// 	// t.Run("QueryChildren", func(t *testing.T) {
// 	// 	tx := s.Txn(false) // read-only tx
// 	// 	defer tx.Commit()

// 	// 	it, err := tx.Get(ref, "path_prefix", "foo", "bar")
// 	// 	assert.NoError(t, err, "should succeed")

// 	// 	var children []string
// 	// 	for v := it.Next(); v != nil; v = it.Next() {
// 	// 		children = append(children, v.(*cluster.Anchor).Name())
// 	// 	}

// 	// 	t.Log(children)

// 	// 	for _, name := range []string{"baz", "qux"} {
// 	// 		assert.Contains(t, children, name,
// 	// 			"query should return all children")
// 	// 	}
// 	// })
// }

// // func insertOne(db *memdb.MemDB, a *cluster.Anchor) error {
// // 	tx := db.Txn(true)
// // 	defer tx.Commit()

// // 	return tx.Insert("anchor", a)
// // }

// func anchor(path string) *cluster.Anchor {
// 	return &cluster.Anchor{
// 		Path: cluster.NewPath(path),
// 	}
// }
