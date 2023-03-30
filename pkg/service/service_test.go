package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	inproc "github.com/lthibault/go-libp2p-inproc-transport"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	api "github.com/wetware/ww/internal/api/service"
	pscap "github.com/wetware/ww/pkg/pubsub"
	"github.com/wetware/ww/pkg/service"
)

func TestDiscover(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gs, release := newGossipSub(ctx)
	defer release()

	ps := (&pscap.Server{TopicJoiner: gs}).PubSub()
	defer ps.Release()

	// create server
	server := service.RegistryServer{}
	// create 1 client
	client := service.Registry(api.Registry_ServerToClient(&server))
	defer client.Release()
	// advertise service in 1 client

	const (
		serviceName = "service.test"
		maddrN      = 2
	)

	// advertise and find
	loc, err := generateLocation(maddrN)
	require.NoError(t, err)

	topic, release := ps.Join(ctx, serviceName)
	defer release()

	_, release = client.Provide(ctx, topic, loc)
	defer release()

	time.Sleep(time.Second) // give time for the provider to set

	providers, release := client.FindProviders(ctx, topic)
	defer release()

	gotLocation, ok := providers.Next()
	require.True(t, ok)

	expected, err := loc.Maddrs()
	require.NoError(t, err)

	got, err := gotLocation.Maddrs()
	require.NoError(t, err)

	require.EqualValues(t, expected, got)
}

func generateLocation(n int) (service.Location, error) {
	loc, err := service.NewLocation()
	if err != nil {
		return loc, fmt.Errorf("failed to create location: %w", err)
	}
	maddrs := make([]ma.Multiaddr, 0, n)

	for i := 0; i < n; i++ {
		maddr, _ := ma.NewMultiaddr("/ip4/0.0.0.0")
		maddrs = append(maddrs, maddr)
	}

	return loc, loc.SetMaddrs(maddrs)
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
