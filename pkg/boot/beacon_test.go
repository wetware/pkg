package boot_test

import (
	"context"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/record"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wetware/ww/pkg/boot"
)

func TestKnock(t *testing.T) {
	t.Parallel()

	k, err := boot.NewKnock("test")
	require.NoError(t, err)
	require.NotZero(t, k.Nonce, "nonce should not be zero")
	require.NotZero(t, k.Hash, "hash should not be zero")

	assert.True(t, k.Matches("test"), "k should match namespace")
}

func TestBeacon(t *testing.T) {
	t.Parallel()

	// Set up peer identity
	pk, _, err := crypto.GenerateECDSAKeyPair(rand.New(rand.NewSource(42)))
	require.NoError(t, err)

	id, err := peer.IDFromPrivateKey(pk)
	require.NoError(t, err)

	e, err := record.Seal(&peer.PeerRecord{
		PeerID: id,
		Addrs: []ma.Multiaddr{
			ma.StringCast("/ip4/10.0.0.128/udp/2020/quic"),
		},
	}, pk)
	require.NoError(t, err)

	// Set up beacon
	var b = boot.Beacon{
		Envelope: e,
		Addr:     "localhost:3021",
	}

	cherr := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		cherr <- b.Serve(ctx)
	}()
	select {
	case err := <-cherr:
		t.Errorf("service aborted: %s", err)
		t.FailNow()
	case <-time.After(time.Millisecond):
	}

	ttl, err := b.Advertise(ctx, "test", discovery.TTL(time.Second))
	require.NoError(t, err)
	assert.Equal(t, time.Second, ttl, "default TTL should have been overridden")

	var s = boot.Scanner{
		Port: 3021,
	}

	var r peer.PeerRecord
	err = s.RoundTrip(ctx, "test", net.IPv4(127, 0, 0, 1), &r)
	require.NoError(t, err)
	assert.Equal(t, id, r.PeerID)

	cancel()
	assert.ErrorIs(t, <-cherr, context.Canceled)
}
