package discovery_test

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	inproc "github.com/lthibault/go-libp2p-inproc-transport"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	api "github.com/wetware/ww/internal/api/discovery"
	"github.com/wetware/ww/pkg/discovery"
	pscap "github.com/wetware/ww/pkg/pubsub"
)

func TestDiscover(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gs, release := newGossipSub(ctx)
	defer release()

	ps := (&pscap.Router{TopicJoiner: gs}).PubSub()
	defer ps.Release()

	// create server
	server := discovery.DiscoveryServiceServer{
		Joiner: ps,
	}
	// create 1 client
	client := discovery.DiscoveryService(api.DiscoveryService_ServerToClient(&server))
	defer client.Release()
	// advertise service in 1 client

	const (
		serviceName = "service.test"
		maddrN      = 2
	)

	// advertise and find
	provider, release := client.Provider(ctx, serviceName)
	defer release()

	addr := generateAddr(maddrN)
	_, release = provider.Provide(ctx, addr)
	defer release()

	time.Sleep(time.Second) // give time for the provider to set

	finder, release := client.Locator(ctx, serviceName)
	defer release()

	providers, release := finder.FindProviders(ctx)
	defer release()

	gotAddr, ok := providers.Next()
	require.True(t, ok)

	require.EqualValues(t, addr, gotAddr)
}

func generateAddr(n int) (addr discovery.Addr) {
	maddrs := make([]ma.Multiaddr, 0, n)

	for i := 0; i < n; i++ {
		maddr, _ := ma.NewMultiaddr("/ip4/0.0.0.0")
		maddrs = append(maddrs, maddr)
	}

	addr.Maddrs = maddrs

	return addr
}

func newGossipSub(ctx context.Context) (*pubsub.PubSub, func()) {
	h := newTestHost()

	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		panic(err)
	}

	return ps, func() { h.Close() }
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
