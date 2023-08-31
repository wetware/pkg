package server_test

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
	"github.com/wetware/pkg/server"
)

func Test(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h, err := libp2p.New(
		libp2p.NoTransports,
		libp2p.NoListenAddrs,
		libp2p.Transport(inproc.New()),
		libp2p.ListenAddrStrings("/inproc/~"))
	require.NoError(t, err)
	defer h.Close()

	dht, err := server.NewDHT(ctx, h, "test")
	require.NoError(t, err)
	defer dht.Close()

	vat := server.Vat{
		NS:        "test",
		Host:      h,
		Bootstrap: nopDiscovery{},
		Ambient:   nopDiscovery{},
		// Auth: ,
	}
	go vat.Serve(ctx)

	term, release := vat.NewTerminal()
	defer release()

	sess, err := term.Login(ctx, nil) // FIXME:  <<-- YOU ARE HERE
	require.NoError(t, err)
	defer sess.Close()

	require.EqualError(t, sess.Err(), "no account specified",
		"auth should fail for nil signer")

	// TODO:  expect errors in all session capabilities

}

type nopDiscovery struct{}

func (nopDiscovery) Advertise(context.Context, string, ...discovery.Option) (time.Duration, error) {
	return peerstore.PermanentAddrTTL, nil
}
func (nopDiscovery) FindPeers(ctx context.Context, _ string, _ ...discovery.Option) (<-chan peer.AddrInfo, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
