package client

import (
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/wetware/ww/pkg/boot"
)

// Option type for Client
type Option func(*Config) error

// WithNamespace sets the cluster namespace to connect to.
func WithNamespace(ns string) Option {
	return func(c *Config) (err error) {
		c.ns = ns
		return
	}
}

// WithDiscover determines how the client will connect to a cluster.
func WithDiscover(d boot.Strategy) Option {
	return func(c *Config) (err error) {
		if d == nil {
			d = boot.MDNS{Namespace: c.ns}
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

func withDefault(opt []Option) []Option {
	return append([]Option{
		WithNamespace("ww"),
		WithDiscover(nil),
		withDataStore(nil),
	}, opt...)
}
