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

// // WithTempNode configures the underlying IFPS node as a short-lived peer, disabling
// // some expensive background processes that improve performance in the long run
// func WithTempNode() Option {
// 	return func(r *Runtime) (err error) {
// 		// Permanent nodes add a layer of caching for block storage (using bloom-
// 		// filters) on top of the standard ARC cache.  Disabling the bloom filter cache
// 		// apparently reduces memory consumption.
// 		//
// 		// The trade-off is that bloom-filter cacheing improves cache latency after an
// 		// initial warm-up period.
// 		r.buildCfg.Permanent = false
// 		return
// 	}
// }

// WithNamespace sets the cluster's namespace
func WithNamespace(ns string) Option {
	return func(r *Runtime) (err error) {
		r.ns = ns
		return
	}
}

// WithClientProfile starts a Host in client-mode.  Client hosts do not store data or
// otherwise participate in cluster activities, but can connect to a cluster.
func WithClientProfile() Option {
	return func(r *Runtime) error {
		return applyOpt(r,
			withProfile(clientProfile))
	}
}

func withProfile(p profile) Option {
	return func(r *Runtime) error {
		r.newBuildCfg = p
		return nil
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
		withProfile(defaultProfile),
	}, opt...)
}

func applyOpt(r *Runtime, opt ...Option) (err error) {
	for _, f := range opt {
		if err = f(r); err != nil {
			break
		}
	}

	return
}
