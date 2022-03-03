package cluster_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	mx "github.com/wetware/matrix/pkg"
	"github.com/wetware/ww/pkg/cap/cluster"
	"github.com/wetware/ww/pkg/vat"
)

func TestContainerWalk(t *testing.T) {
	t.Parallel()
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sim := mx.New(ctx)
	hs := sim.MustHostSet(ctx, 2)

	vat0 := vat.Network{NS: "test-container", Host: hs[0]}
	server := cluster.NewHostAnchorServer(vat0, nil)
	client := server.NewClient()

	a1, err := client.Walk(ctx, []string{"foo"})
	require.NoError(t, err)
	require.NotNil(t, a1)
	expectedPath := []string{hs[0].ID().String(), "foo"}
	require.Equal(t, expectedPath, a1.Path())

	a2, err := a1.Walk(ctx, []string{"bar"})
	require.NoError(t, err)
	require.NotNil(t, a2)
	expectedPath = []string{hs[0].ID().String(), "foo", "bar"}
	require.Equal(t, expectedPath, a2.Path())
}

func TestContainerLs(t *testing.T) {
	t.Parallel()
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sim := mx.New(ctx)
	hs := sim.MustHostSet(ctx, 2)

	vat0 := vat.Network{NS: "test-container", Host: hs[0]}
	server := cluster.NewHostAnchorServer(vat0, nil)
	client := server.NewClient()

	a1, err := client.Walk(ctx, []string{"foo"})
	require.NoError(t, err)
	_, err = a1.Walk(ctx, []string{"bar"})
	require.NoError(t, err)

	expectedPath := []string{hs[0].ID().String(), "foo", "bar"}

	it, err := a1.Ls(ctx)
	require.NoError(t, err)
	require.NotNil(t, it)
	require.True(t, it.Next(ctx))
	require.Equal(t, expectedPath, it.Anchor().Path())
	require.False(t, it.Next(ctx))
	require.Nil(t, it.Err())
}

func TestContainerGetSet(t *testing.T) {
	t.Parallel()
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sim := mx.New(ctx)
	hs := sim.MustHostSet(ctx, 2)

	vat0 := vat.Network{NS: "test-container", Host: hs[0]}
	server := cluster.NewHostAnchorServer(vat0, nil)
	client := server.NewClient()

	a1, err := client.Walk(ctx, []string{"foo"})
	require.NoError(t, err)

	c1, ok := a1.(cluster.Container)
	require.True(t, ok)
	require.NotNil(t, c1)

	data, release := c1.Get(ctx)
	defer release()
	require.NoError(t, err)
	require.Nil(t, data)

	err = c1.Set(ctx, []byte("test-container"))
	require.NoError(t, err)
	data, release = c1.Get(ctx)
	defer release()
	require.NoError(t, err)
	require.EqualValues(t, []byte("test-container"), data)
}
