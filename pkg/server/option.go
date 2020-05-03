package server

import (
	"time"

	"github.com/libp2p/go-libp2p-core/pnet"
	log "github.com/lthibault/log/pkg"
	ww "github.com/lthibault/wetware/pkg"
	discover "github.com/lthibault/wetware/pkg/discover"
	"github.com/multiformats/go-multiaddr"
)

// Option type for Host
type Option func(*Config) error

// Config .
type Config struct {
	log   log.Logger
	trace bool

	ns         string
	ttl        time.Duration
	kmin, kmax int // min, max node cardinality

	addrs []multiaddr.Multiaddr
	psk   pnet.PSK

	boot discover.Protocol
}

/*
	Options
*/

// WithLogger sets the logger.
func WithLogger(logger log.Logger) Option {
	return func(c *Config) (err error) {
		c.log = logger
		return
	}
}

// WithEventTrace controls the logging of events in the Host's internal bus.
func WithEventTrace(trace bool) Option {
	return func(c *Config) (err error) {
		c.trace = trace
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

// WithDiscover sets the Host's bootstrap strategy.
func WithDiscover(p discover.Protocol) Option {
	return func(c *Config) (err error) {
		if p == nil {
			p = &discover.MDNS{Namespace: c.ns}
		}

		c.boot = p
		return
	}
}

// WithTTL specifies the TTL for the heartbeat protocol.  `0` specifies a default value
// of 6 seconds, which is suitable for almost all applications.
//
// The most common reason to adjust the TTL is in testing, where it may be desireable to
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

/*
	Utils
*/

func withDefault(opt []Option) []Option {
	return append([]Option{
		WithLogger(log.New(log.OptLevel(log.FatalLevel))),
		WithEventTrace(false),
		WithNamespace(ww.DefaultNamespace),
		WithListenAddrString(
			"/ip4/127.0.0.1/tcp/0", // IPv4 loopback
			"/ip6/::1/tcp/0",       // IPv6 loopback
		),
		WithDiscover(nil),
		WithTTL(0),
		withCardinality(8, 32),
	}, opt...)
}
