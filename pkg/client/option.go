package client

import (
	"github.com/lthibault/log"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"

	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/boot"
)

// Option type for Client
type Option func(*Config) error

// WithLogger sets the client logger
func WithLogger(l ww.Logger) Option {
	if l == nil {
		l = log.New()
	}

	return func(c *Config) (err error) {
		c.log = l
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

// WithStrategy determines how the client will connect to a cluster.
func WithStrategy(d boot.Strategy) Option {
	return func(c *Config) (err error) {
		if d == nil {
			d = boot.MDNS{Namespace: c.ns}
		}

		c.d = d
		return
	}
}

func withCardinality(k, highwater int) Option {
	return func(c *Config) (err error) {
		c.kmin = k
		c.kmax = highwater
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

func withDefault(opt []Option) []Option {
	return append([]Option{
		WithLogger(nil),
		WithNamespace("ww"),
		WithStrategy(nil),
		withCardinality(3, 64),
		withDataStore(nil),
	}, opt...)
}
