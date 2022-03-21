package cluster_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/pkg/cap/cluster"
	"github.com/wetware/ww/pkg/vat"
)

func TestAnchor(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := cluster.NewHost(vat.Network{})

	h := cluster.Host{
		Client: s.Client(),
	}

	t.Run("Empty", func(t *testing.T) {
		rs, release := h.Ls(ctx)
		require.NotNil(t, rs, "should return register set")
		require.NotNil(t, release, "should return release function")
		defer release()

		assert.False(t, rs.More(), "should have zero children")
	})

	t.Run("NotEmpty", func(t *testing.T) {
		path := []string{"alpha"}

		r, release := h.Walk(ctx, path)
		require.NotZero(t, r, "should return register")
		require.NotNil(t, release, "should return release function")
		defer release()

		rs, release := h.Ls(ctx)
		require.NotNil(t, rs, "should return register set")
		require.NotNil(t, release, "should return release function")
		defer release()

		ss, err := toSlice(rs)
		require.NoError(t, err, "should iterate without error")
		assert.Equal(t, path, ss)
	})

	t.Run("MultiLevel/Empty", func(t *testing.T) {
		path := []string{"alpha", "bravo"}

		r, release := h.Walk(ctx, path)
		require.NotZero(t, r, "should return register")
		require.NotNil(t, release, "should return release function")
		defer release()

		// root should have child 'alpha'
		rs, release := h.Ls(ctx)
		require.NotNil(t, rs, "should return register set")
		require.NotNil(t, release, "should return release function")
		defer release()

		ss, err := toSlice(rs)
		require.NoError(t, err, "should iterate without error")
		assert.Equal(t, []string{"alpha"}, ss)

		// alpha should have child 'bravo'
		r, release = h.Walk(ctx, []string{"alpha"})
		defer release()

		rs, release = r.Ls(ctx)
		defer release()

		ss, err = toSlice(rs)
		require.NoError(t, err, "should iterate without error")
		assert.Equal(t, []string{"bravo"}, ss)
	})

	t.Run("AnchorRelease", func(t *testing.T) {
		runtime.GC()

		rs, release := h.Ls(ctx)
		require.NotNil(t, rs, "should return register set")
		require.NotNil(t, release, "should return release function")
		defer release()

		assert.False(t, rs.More(), "should have zero children")
	})
}

func toSlice(rs *cluster.RegisterMap) ([]string, error) {
	var ss []string
	for rs.Next() {
		ss = append(ss, rs.Name)
	}

	return ss, rs.Err
}
