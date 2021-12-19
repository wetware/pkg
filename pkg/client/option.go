package client

import (
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/lthibault/log"
)

type Option func(*Dialer)

// WithNamespace sets the cluster namespace used by the overlay.
// If ns == "", the default namespace "ww" is used.
func WithNamespace(ns string) Option {
	if ns == "" {
		ns = "ww"
	}

	return func(d *Dialer) {
		d.ns = ns
	}
}

// WithLogger sets the logger for the client node.  If n == nil,
// a default logger is used.
func WithLogger(l log.Logger) Option {
	if l == nil {
		l = log.New()
	}

	return func(d *Dialer) {
		d.log = l
	}
}

// WithHost specifies the host used by the dialer.  If h == nil, each
// call to 'Dial' will create a new client host.
//
// Users are responsible for closing 'h' when finished, and are advised
// that calls to 'Node.Close' will implicitly close 'h'.
//
// In most cases, 'h' SHOULD NOT listen for incoming connections.
func WithHost(h host.Host) Option {
	return func(d *Dialer) {
		d.host = h
	}
}

// WithRouting configures the client's routing implementation.
// If r == nil, a default DHT client is used.
func WithRouting(r RoutingFactory) Option {
	if r == nil {
		r = DefaultRouting
	}

	return func(d *Dialer) {
		d.newRouting = r
	}
}

// WithPubSub sets the pubsub instance used to construct the overlay.
// This instance MUST be bound to 'h'.  Passing 'WithPubSub' without
// a corresponding 'WithHost' causes undefined behavior.  Use caution.
func WithPubSub(p PubSub) Option {
	return func(d *Dialer) {
		d.pubsub = p
	}
}

func withDefault(opt []Option) []Option {
	return append([]Option{
		WithNamespace(""),
		WithLogger(nil),
		WithHost(nil),
		WithRouting(nil),
		WithPubSub(nil),
	}, opt...)
}
