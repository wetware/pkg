package host

import (
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/lthibault/log"

	"github.com/multiformats/go-multiaddr"

	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/boot"
)

// Option type for Host
type Option func(*Config) error

// WithLogger sets the default logger for the Host.
func WithLogger(l ww.Logger) Option {
	if l == nil {
		l = log.New()
	}

	return func(c *Config) (err error) {
		c.log = l
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

// WithListenAddrString sets the Host's listen address(es).  Panics if string is not a
// valid multiaddr.
func WithListenAddrString(addrs ...string) Option {
	return func(c *Config) (err error) {
		as := make([]multiaddr.Multiaddr, len(addrs))
		for i, s := range addrs {
			if as[i], err = multiaddr.NewMultiaddr(s); err != nil {
				break
			}
		}

		c.addrs = as
		return
	}
}

// WithListenAddr sets the Host's listen address(es).
func WithListenAddr(addrs ...multiaddr.Multiaddr) Option {
	return func(c *Config) (err error) {
		c.addrs = addrs
		return
	}
}

// WithBootStrategy sets the Host's bootstrap strategy.  Nil configures a default using
// MDNS.
func WithBootStrategy(b boot.Strategy) Option {
	return func(c *Config) (err error) {
		if b == nil {
			b = &boot.MDNS{Namespace: c.ns}
		}

		c.boot = b
		return
	}
}

// WithTTL specifies the TTL for the heartbeat protocol.  `0` specifies a default value
// of 6 seconds, which is suitable for almost all applications.
//
// The most common reason to adjust the TTL is in testing, where it may be desirable to
// reduce the time needed for peers to become mutually aware.
func WithTTL(ttl time.Duration) Option {
	if ttl == 0 {
		ttl = time.Second * 6
	}

	return func(c *Config) (err error) {
		c.ttl = ttl
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
		WithNamespace(ww.DefaultNamespace),
		WithListenAddrString(
			"/ip4/127.0.0.1/tcp/0", // IPv4 loopback
			"/ip6/::1/tcp/0",       // IPv6 loopback
		),
		WithBootStrategy(nil),
		WithTTL(0),
		withCardinality(8, 32),
		withDataStore(nil),
	}, opt...)
}
