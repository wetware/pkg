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

	server := ww.HostServer{ViewProvider: vp}
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
func (mockRecord) Instance() routing.ID        { return routing.ID(42) }
func (mockRecord) Seq() uint64                 { return 1 }
func (mockRecord) Host() (string, error)       { return "foo", nil }
func (mockRecord) Meta() (routing.Meta, error) { return routing.Meta{}, nil }
