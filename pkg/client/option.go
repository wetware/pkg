package client

import (
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	log "github.com/lthibault/log/pkg"
	discover "github.com/lthibault/wetware/pkg/discover"

	"github.com/libp2p/go-libp2p-core/pnet"
)

// Option type for Client
type Option func(*Config) error

// Config .
type Config struct {
	log log.Logger
	ns  string
	psk pnet.PSK
	ds  datastore.Batching

	d          discover.Strategy
	queryLimit int
}

// WithLogger sets the logger.
func WithLogger(logger log.Logger) Option {
	return func(c *Config) (err error) {
		c.log = logger
		return
	}
}

// WithNamespace sets the cluster namespace to connect to.
func WithNamespace(ns string) Option {
	return func(c *Config) (err error) {
		c.ns = ns
		return
	}
}

// WithDiscover determines how the client will connect to a cluster.
func WithDiscover(d discover.Strategy) Option {
	return func(c *Config) (err error) {
		if d == nil {
			d = discover.MDNS{Namespace: c.ns}
		}

		c.d = d
		return
	}
}

func withDataStore(d datastore.Batching) Option {
	if d == nil {
		d = sync.MutexWrap(datastore.NewMapDatastore())
	}

	return func(c *Config) (err error) {
		c.ds = d
		return
	}
}

func withLimit(lim int) Option {
	return func(c *Config) (err error) {
		c.queryLimit = lim
		return
	}
}

func withDefault(opt []Option) []Option {
	return append([]Option{
		WithLogger(log.New(log.OptLevel(log.FatalLevel))),
		WithNamespace("ww"),
		WithDiscover(nil),
		withDataStore(nil),
		withLimit(1),
	}, opt...)
}
