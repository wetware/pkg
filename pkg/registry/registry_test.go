package service_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/record"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	inproc "github.com/lthibault/go-libp2p-inproc-transport"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	api "github.com/wetware/ww/internal/api/registry"
	pscap "github.com/wetware/ww/pkg/pubsub"
	service "github.com/wetware/ww/pkg/registry"
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
	loc, err := generateLocation(maddrN, serviceName)
	require.NoError(t, err)

	privKey, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
	require.NoError(t, err)

	e, err := record.Seal(&loc, privKey)
	require.NoError(t, err)

	topic, release := ps.Join(ctx, serviceName)
	defer release()

	_, release = client.Provide(ctx, topic, e)
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

func generateLocation(n int, serviceName string) (service.Location, error) {
	loc, err := service.NewLocation()
	if err != nil {
		return loc, fmt.Errorf("failed to create location: %w", err)
	}
	maddrs := make([]ma.Multiaddr, 0, n)

	for i := 0; i < n; i++ {
		maddr, _ := ma.NewMultiaddr("/ip4/0.0.0.0")
		maddrs = append(maddrs, maddr)
	}

	if err := loc.SetMaddrs(maddrs); err != nil {
		return loc, fmt.Errorf("failed to set maddrs: %w", err)
	}

	if err := loc.SetService(serviceName); err != nil {
		return loc, fmt.Errorf("failed to set service name: %w", err)
	}

	return loc, nil
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
