package ww

import (
	"time"

	log "github.com/lthibault/log/pkg"
)

// Option type for Host
type Option func(*Runtime) error

// WithLogger sets the logger.
func WithLogger(logger log.Logger) Option {
	return func(r *Runtime) (err error) {
		r.log = logger
		return
	}
}

// WithNamespace sets the cluster's namespace
func WithNamespace(ns string) Option {
	return func(r *Runtime) (err error) {
		r.ns = ns
		return
	}
}

func withTTL(ttl time.Duration) Option {
	return func(r *Runtime) (err error) {
		r.ttl = ttl
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
