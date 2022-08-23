package runtime

import (
	"github.com/libp2p/go-libp2p"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/pex"
)

// Config is used to parametrize the Fx runtime. It contains
// a set of unexported type-constructors that are set by the
// Option type, and then provided to Fx. This allows callers
// to override defaults, while keeping interacton with Fx to
// a minimum.
type Config struct {
	newHost HostConfig
	hostOpt []libp2p.Option

	pexOpt []pex.Option
}

// Option can modify the state of Config.  It is used to set
// type constructors that will be consumed by Fx.
type Option func(*Config)

// HostConfig specifies how to construct a libp2p Host, in a
// parametrizable way. Implementations MUST pass the options
// provided to libp2p.New.  They MAY prepend default options.
type HostConfig func(...libp2p.Option) casm.HostFactory

// WithHostConfig sets the host configuration for the Fx app.
// Panics if f == nil.
func WithHostConfig(f HostConfig) Option {
	if f == nil {
		panic("HostConfig(nil)")
	}

	return func(c *Config) {
		c.newHost = f
	}
}

// HostOpt declares a set of libp2p options to be passed into
// the HostConifg.  If len(opt) == nil, no options are passed.
func HostOpt(opt ...libp2p.Option) Option {
	if len(opt) == 0 {
		opt = nil
	}

	return func(c *Config) {
		c.hostOpt = opt
	}
}

// WithPeXOpt sets the options for the boot cache.  The PeX cache is
// disabled by default.   Calling without arguments uses the default
// configuration.
func WithPeXOpt(opt ...pex.Option) Option {
	return func(c *Config) {
		c.pexOpt = opt
	}
}

// WithPeXDisabled disables PeX entirely.
func WithPeXDisabled() Option {
	return func(c *Config) {
		c.pexOpt = nil
	}
}
