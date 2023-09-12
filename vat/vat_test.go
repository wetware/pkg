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
	"github.com/wetware/pkg/vat"
)

func TestServe(t *testing.T) {
	t.Parallel()

	const d = time.Millisecond * 10
	ctx, cancel := context.WithTimeout(context.Background(), d)
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

	cherr := make(chan error, 1)
	go func() {
		cherr <- vat.Config{
			NS:        "test",
			Host:      h,
			Bootstrap: nopDiscovery{},
			Ambient:   nopDiscovery{},
			Auth:      auth.AllowAll,
			// OnLogin: func(root auth.Session) {
			// 	defer cancel()
			// 	require.NotZero(t, root, "must return non-null Host")
			// },
		}.Serve(ctx)
	}()
	require.ErrorIs(t, <-cherr, context.DeadlineExceeded)
}

type nopDiscovery struct{}

func (nopDiscovery) Advertise(context.Context, string, ...discovery.Option) (time.Duration, error) {
	return peerstore.PermanentAddrTTL, nil
}
func (nopDiscovery) FindPeers(ctx context.Context, _ string, _ ...discovery.Option) (<-chan peer.AddrInfo, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
