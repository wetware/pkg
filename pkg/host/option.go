package host

import (
	"time"

	log "github.com/lthibault/log/pkg"
)

// Option type for Host
type Option func(*Config) error

// Config .
type Config struct {
	log log.Logger
	ns  string
	ttl time.Duration

	repoPath string // TODO:  refactor repository construction
}

func newConfig(opt []Option) (*Config, error) {
	cfg := new(Config)
	for _, f := range withDefault(opt) {
		if err := f(cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// Log returns a logger with attached fields. Prefer this to using cfg.log directly.
func (cfg Config) Log() log.Logger {
	return cfg.log.WithField("ns", cfg.ns)
}

// WithLogger sets the logger.
func WithLogger(logger log.Logger) Option {
	return func(c *Config) (err error) {
		c.log = logger
		return
	}
}

// WithNamespace sets the cluster's namespace
func WithNamespace(ns string) Option {
	return func(c *Config) (err error) {
		c.ns = ns
		return
	}
}

func withTTL(ttl time.Duration) Option {
	return func(c *Config) (err error) {
		c.ttl = ttl
		return
	}
}

func withDefault(opt []Option) []Option {
	return append([]Option{
		WithLogger(log.New(log.OptLevel(log.FatalLevel))),
		WithNamespace("ww"),
		withTTL(time.Second * 6),
	}, opt...)
}
