package client

import (
	"github.com/libp2p/go-libp2p"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	"github.com/libp2p/go-libp2p/config"
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

func WithHostOpts(opt ...config.Option) Option {
	if len(opt) == 0 {
		opt = append(opt,
			libp2p.NoListenAddrs,
			libp2p.NoTransports,
			libp2p.Transport(libp2pquic.NewTransport))
	}

	return func(d *Dialer) {
		d.hostOpts = opt
	}
}

func withDefault(opt []Option) []Option {
	return append([]Option{
		WithNamespace(""),
		WithLogger(nil),
		WithHostOpts(),
	}, opt...)
}
