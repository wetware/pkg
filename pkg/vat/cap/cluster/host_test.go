package cluster_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/pkg/vat/cap/anchor"
	"github.com/wetware/ww/pkg/vat/cap/cluster"
)

func TestLs(t *testing.T) {
	t.Parallel()

	/*
		This is a simple test that asserts a new host has no children.
	*/

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := cluster.Host{
		// pre-resolved; can pass nil Dialer/MergeStrategy to methods.
		Client: new(cluster.HostServer).Client(),
	}

	cs, release := h.Ls(ctx, nil)
	defer release()

	assert.False(t, cs.Next(), "fresh host should not contain children")
}

func TestHost_Walk(t *testing.T) {
	t.Parallel()

	/*
		This is a simple test that asserts a new host can walk to an
		arbitrary path, that the resulting cluster has no children,
		and that the host has the expected number of children.
	*/

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := cluster.Host{
		// pre-resolved; can pass nil Dialer/MergeStrategy to methods.
		Client: new(cluster.HostServer).Client(),
	}

	// Walk to /foo/bar
	bar, release := h.Walk(ctx, nil, anchor.NewPath("/foo/bar"))
	defer release()

	// Check that bar has no children
	bcs, release := bar.Ls(ctx)
	defer release()

	assert.False(t, bcs.Next(), "node 'bar' should not have children")

	// Check that host has a single child
	hcs, release := h.Ls(ctx, nil)
	defer release()

	var children []string
	for hcs.Next() {
		children = append(children, hcs.Name)
	}

	require.Len(t, children, 1, "root node have exactly one child")
	require.Contains(t, children, "foo", "node 'foo' should be child of root")

	// Check that foo has a single child
	foo, release := h.Walk(ctx, nil, anchor.NewPath("/foo"))
	defer release()

	fcs, release := foo.Ls(ctx)
	defer release()

	children = children[:0]
	for fcs.Next() {
		children = append(children, fcs.Name)
	}

	require.Len(t, children, 1, "node 'foo' should have exactly one child")
	require.Contains(t, children, "bar", "node 'bar' should be child of foo")
}
