package vat_test

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	inproc "github.com/lthibault/go-libp2p-inproc-transport"
	"github.com/stretchr/testify/require"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/vat"
)

func TestServer(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h, err := libp2p.New(
		libp2p.NoTransports,
		libp2p.NoListenAddrs,
		libp2p.Transport(inproc.New()),
		libp2p.ListenAddrStrings("/inproc/~"))
	require.NoError(t, err)
	defer h.Close()

	dht, err := vat.NewDHT(ctx, h, "test")
	require.NoError(t, err)
	defer dht.Close()

	err = vat.Config{
		NS:        "test",
		Host:      h,
		Bootstrap: nopDiscovery{},
		Ambient:   nopDiscovery{},
		Auth:      auth.AllowAll,
		OnJoin: func(root host.Host) {
			defer cancel()
			defer root.Release()

			require.NotZero(t, root, "must return non-null Host")
		},
	}.Serve(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

type nopDiscovery struct{}

func (nopDiscovery) Advertise(context.Context, string, ...discovery.Option) (time.Duration, error) {
	return peerstore.PermanentAddrTTL, nil
}
func (nopDiscovery) FindPeers(ctx context.Context, _ string, _ ...discovery.Option) (<-chan peer.AddrInfo, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
