package host

import (
	"testing"

	"github.com/lthibault/wetware/pkg/boot"
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
		assert.NotNil(t, cfg.boot,
			"no discovery service supplied by default")
		assert.Equal(t, &boot.MDNS{Namespace: cfg.ns}, cfg.boot,
			"unexpected default discovery service")
	})
}

func TestDiscoveryOpt(t *testing.T) {
	cfg := Config{ns: "test"}

	// default is MDNS using most recent namespace value
	t.Run("DefaultUsesNamespace", func(t *testing.T) {
		require.NoError(t, WithBootStrategy(nil)(&cfg),
			"unable to auto-assign discovery service")
		assert.NotNil(t, cfg.boot,
			"no discovery service supplied by default")
		assert.Equal(t, &boot.MDNS{Namespace: "test"}, cfg.boot,
			"unexpected default discovery service")
	})

	t.Run("Override", func(t *testing.T) {
		require.NoError(t, WithBootStrategy(boot.StaticAddrs{})(&cfg),
			"unable to override discovery service")
		assert.Equal(t, boot.StaticAddrs{}, cfg.boot,
			"config does not contain override value")
	})
}
