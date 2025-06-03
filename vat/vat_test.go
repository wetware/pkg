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
	core_api "github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/auth"
	csp_server "github.com/wetware/pkg/cap/csp/server"
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

	ec := make(chan csp_server.Runtime, 1)
	sc := make(chan core_api.Session, 1)
	err = vat.Config{
		NS:        "test",
		Host:      h,
		Bootstrap: nopDiscovery{},
		Ambient:   nopDiscovery{},
		Auth:      auth.AllowAll,
		OnJoin: func(root auth.Session) {
			defer cancel()
			require.NotZero(t, root, "must return non-null Host")
		},
	}.Serve(ctx, ec, sc, h)
	<-sc
	<-ec
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
