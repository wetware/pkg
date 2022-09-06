package ww_test

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/casm/pkg/cluster/routing"
	mock_ww "github.com/wetware/ww/internal/mock/pkg"
	ww "github.com/wetware/ww/pkg"
)

func TestHost_View(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	rt := mockRoutingTable{}
	vs := cluster.Server{RoutingTable: rt}

	vp := mock_ww.NewMockViewProvider(ctrl)
	vp.EXPECT().
		View().
		Return(cluster.View(vs.Client())).
		Times(1)

	server := ww.HostServer{Cluster: vp}
	v, release := server.Host().View(context.Background())
	require.NotNil(t, release)
	defer release()
	require.NotZero(t, v)

	f, release := v.Lookup(context.Background(), cluster.All())
	defer release()
	r, err := f.Await(context.Background())
	require.NoError(t, err)
	require.NotNil(t, r)
}

type mockRoutingTable struct{}

func (mockRoutingTable) Snapshot() routing.Snapshot {
	return mockSnapshot{}
}

type mockSnapshot struct{}

func (mockSnapshot) Get(routing.Index) (routing.Iterator, error) {
	it := make(mockIterator, 1)
	return &it, nil
}
func (mockSnapshot) GetReverse(routing.Index) (routing.Iterator, error) {
	it := make(mockIterator, 1)
	return &it, nil
}
func (mockSnapshot) LowerBound(routing.Index) (routing.Iterator, error) {
	it := make(mockIterator, 1)
	return &it, nil
}
func (mockSnapshot) ReverseLowerBound(routing.Index) (routing.Iterator, error) {
	it := make(mockIterator, 1)
	return &it, nil
}

type mockIterator []mockRecord

func (it *mockIterator) Next() routing.Record {
	switch len(*it) {
	case 0:
		return nil

	case 1:
		r := (*it)[0]
		*it = nil
		return r

	default:
		r := (*it)[0]
		*it = (*it)[1:]
		return r
	}
}

type mockRecord struct{}

func (mockRecord) Peer() peer.ID               { return "foobar" }
func (mockRecord) TTL() time.Duration          { return time.Second }
func (mockRecord) Seq() uint64                 { return 1 }
func (mockRecord) Instance() uint32            { return 42 }
func (mockRecord) Host() (string, error)       { return "foo", nil }
func (mockRecord) Meta() (routing.Meta, error) { return routing.Meta{}, nil }

// func TestLs(t *testing.T) {
// 	t.Parallel()

// 	/*
// 		This is a simple test that asserts a new host has no children.
// 	*/

// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	h := cluster.Host{
// 		// pre-resolved; can pass nil Dialer/MergeStrategy to methods.
// 		Client: new(cluster.HostServer).Client(),
// 	}

// 	cs, release := h.Ls(ctx, nil)
// 	defer release()

// 	assert.False(t, cs.Next(), "fresh host should not contain children")
// }

// func TestHost_Walk(t *testing.T) {
// 	t.Parallel()

// 	/*
// 		This is a simple test that asserts a new host can walk to an
// 		arbitrary path, that the resulting cluster has no children,
// 		and that the host has the expected number of children.
// 	*/

// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	h := cluster.Host{
// 		// pre-resolved; can pass nil Dialer/MergeStrategy to methods.
// 		Client: new(cluster.HostServer).Client(),
// 	}

// 	// Walk to /foo/bar
// 	bar, release := h.Walk(ctx, nil, anchor.NewPath("/foo/bar"))
// 	defer release()

// 	// Check that bar has no children
// 	bcs, release := bar.Ls(ctx)
// 	defer release()

// 	assert.False(t, bcs.Next(), "node 'bar' should not have children")

// 	// Check that host has a single child
// 	hcs, release := h.Ls(ctx, nil)
// 	defer release()

// 	var children []string
// 	for hcs.Next() {
// 		children = append(children, hcs.Name)
// 	}

// 	require.Len(t, children, 1, "root node have exactly one child")
// 	require.Contains(t, children, "foo", "node 'foo' should be child of root")

// 	// Check that foo has a single child
// 	foo, release := h.Walk(ctx, nil, anchor.NewPath("/foo"))
// 	defer release()

// 	fcs, release := foo.Ls(ctx)
// 	defer release()

// 	children = children[:0]
// 	for fcs.Next() {
// 		children = append(children, fcs.Name)
// 	}

// 	require.Len(t, children, 1, "node 'foo' should have exactly one child")
// 	require.Contains(t, children, "bar", "node 'bar' should be child of foo")
// }
