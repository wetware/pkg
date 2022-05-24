package stm_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/pkg/cap/cluster/internal/stm"
)

func TestVisitor(t *testing.T) {
	t.Parallel()

	t.Run("Trace/NilRoot/EmptyPath", func(t *testing.T) {
		t.Parallel()

		var (
			root      *stm.Node
			path      stm.StringSliceIterator
			want, got []string
		)

		trace := stm.Visitor(func(n *stm.Node) (stop bool) {
			got = append(got, n.Name)
			return
		})

		err := trace(stm.NodeIterator{
			Current: root,
			Path:    path,
		})

		require.NoError(t, err, "trace should succeed")
		require.Equal(t, want, got, "unexpected trace")
	})

	t.Run("TraceEmpty/NilRoot", func(t *testing.T) {
		t.Parallel()

		var (
			root      *stm.Node
			path      = stm.StringSliceIterator{"alpha"}
			want, got []string
		)

		trace := stm.Visitor(func(n *stm.Node) (stop bool) {
			got = append(got, n.Name)
			return
		})

		err := trace(stm.NodeIterator{
			Current: root,
			Path:    path,
		})

		require.NoError(t, err, "trace should succeed")
		require.Equal(t, want, got, "unexpected trace")
	})
}

// func TestPath(t *testing.T) {
// 	t.Parallel()
// 	t.Helper()

// 	t.Run("NewRootPath", func(t *testing.T) {
// 		t.Parallel()
// 		t.Helper()

// 		for _, tt := range []struct {
// 			name       string
// 			root, want *stm.Node
// 			path       stm.StringSliceIterator
// 		}{
// 			{
// 				name: "Empty_nop",
// 			},
// 		} {
// 			t.Run(tt.name, func(t *testing.T) {
// 				got, err := stm.Walk(stm.NodeIterator{
// 					Current: tt.root,
// 					Path:    tt.path,
// 				})

// 				assert.NoError(t, err, "%s should succeed")
// 				assert.Equal(t, tt.want, got, "%s produced unexpected node")
// 			})
// 		}
// 	})
// }
