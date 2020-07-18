package routing_test

import (
	"testing"
	"time"

	"github.com/lthibault/wetware/pkg/routing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHeartbeat(t *testing.T) {
	ttl := time.Second * 5

	hb, err := routing.NewHeartbeat(ttl)
	require.NoError(t, err)

	assert.Equal(t, ttl, hb.TTL())
}
