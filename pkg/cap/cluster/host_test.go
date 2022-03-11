package cluster_test

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	inproc "github.com/lthibault/go-libp2p-inproc-transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/pkg/cap/cluster"
	"github.com/wetware/ww/pkg/vat"
)

func TestHostWalk(t *testing.T) {
	t.Parallel()
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hs := makeHosts(2)
	closeAll(t, hs)

	vat := vat.Network{NS: "test-host", Host: hs[0]}
	server := cluster.NewHostAnchorServer(vat)
	vat.Export(cluster.AnchorCapability, server)

	client := server.NewClient()

	a1, err := client.Walk(ctx, []string{"foo"})
	require.NoError(t, err)
	require.NotNil(t, a1)
	expectedPath := []string{hs[0].ID().String(), "foo"}
	require.Equal(t, expectedPath, a1.Path())
}

func TestHostLs(t *testing.T) {
	t.Parallel()
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hs := makeHosts(2)
	closeAll(t, hs)

	vat := vat.Network{NS: "test-host", Host: hs[0]}
	server := cluster.NewHostAnchorServer(vat)
	vat.Export(cluster.AnchorCapability, server)

	client := server.NewClient()

	_, err := client.Walk(ctx, []string{"foo"})
	require.NoError(t, err)
	expectedPath := []string{hs[0].ID().String(), "foo"}

	it, err := client.Ls(ctx)
	require.NoError(t, err)
	require.NotNil(t, it)
	require.True(t, it.Next(ctx))
	require.Equal(t, expectedPath, it.Anchor().Path())
	require.False(t, it.Next(ctx))
	require.Nil(t, it.Err())
}

func closeAll(t *testing.T, hs []host.Host) {
	hmap(hs, func(i int, h host.Host) error {
		assert.NoError(t, h.Close(), "should shutdown gracefully (index=%d)", i)
		return nil
	})
}

func hmap(hs []host.Host, f func(i int, h host.Host) error) (err error) {
	for i, h := range hs {
		if err = f(i, h); err != nil {
			break
		}
	}
	return
}

func makeHosts(n int) []host.Host {
	hs := make([]host.Host, n)
	for i := range hs {
		hs[i] = newTestHost()
	}
	return hs
}

func newTestHost() host.Host {
	h, err := libp2p.New(
		libp2p.NoListenAddrs,
		libp2p.NoTransports,
		libp2p.Transport(inproc.New()),
		libp2p.ListenAddrStrings("/inproc/~"))
	if err != nil {
		panic(err)
	}

	return h
}
