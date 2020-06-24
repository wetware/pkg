package routing_test

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/lthibault/wetware/pkg/internal/routing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHeartbeat(t *testing.T) {
	ttl := time.Second * 5

	id, err := peer.Decode("QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N")
	require.NoError(t, err)

	hb, err := routing.NewHeartbeat(id, ttl)
	require.NoError(t, err)

	assert.Equal(t, id, hb.ID())
	assert.Equal(t, ttl, hb.TTL())
}
