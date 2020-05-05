package server

import (
	"testing"

	discover "github.com/lthibault/wetware/pkg/discover"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultOpt(t *testing.T) {
	var cfg Config
	for i, f := range withDefault([]Option{}) {
		require.NoError(t, f(&cfg),
			"error applying default option %d", i)
	}

	t.Run("Namespace", func(t *testing.T) {
		assert.Equal(t, "ww", cfg.ns,
			"invalid default cluster namespace")
	})

	t.Run("Discover", func(t *testing.T) {
		assert.NotNil(t, cfg.d,
			"no discovery service supplied by default")
		assert.Equal(t, &discover.MDNS{Namespace: cfg.ns}, cfg.d,
			"unexpected default discovery service")
	})
}

func TestDiscoveryOpt(t *testing.T) {
	cfg := Config{ns: "test"}

	// default is MDNS using most recent namespace value
	t.Run("DefaultUsesNamespace", func(t *testing.T) {
		require.NoError(t, WithDiscover(nil)(&cfg),
			"unable to auto-assign discovery service")
		assert.NotNil(t, cfg.d,
			"no discovery service supplied by default")
		assert.Equal(t, &discover.MDNS{Namespace: "test"}, cfg.d,
			"unexpected default discovery service")
	})

	t.Run("Override", func(t *testing.T) {
		require.NoError(t, WithDiscover(discover.StaticAddrs{})(&cfg),
			"unable to override discovery service")
		assert.Equal(t, discover.StaticAddrs{}, cfg.d,
			"config does not contain override value")
	})
}
