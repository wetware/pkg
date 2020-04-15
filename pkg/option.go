package ww

import (
	"time"

	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
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

// WithTempNode configures the underlying IFPS node as a short-lived peer, disabling
// some expensive background processes that improve performance in the long run
func WithTempNode() Option {
	return func(r *Runtime) (err error) {
		// Permanent nodes add a layer of caching for block storage (using bloom-
		// filters) on top of the standard ARC cache.  Disabling the bloom filter cache
		// apparently reduces memory consumption.
		//
		// The trade-off is that bloom-filter cacheing improves cache latency after an
		// initial warm-up period.
		r.buildCfg.Permanent = false
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

// WithClientMode starts a Host in client-mode.  Client hosts do not store data or
// otherwise participate in cluster activities, but can connect to a cluster.
func WithClientMode() Option {
	return func(r *Runtime) error {
		r.clientMode = true
		return applyOpt(r,
			WithTempNode(),
			withDHTClient(),
			withNilRepo())
	}
}

func withTTL(ttl time.Duration) Option {
	return func(r *Runtime) (err error) {
		r.ttl = ttl
		return
	}
}

func withBuildConfig(cfg *core.BuildCfg) Option {
	if cfg == nil {
		cfg = defaultBuildConfig()
	}

	return func(r *Runtime) (err error) {
		r.buildCfg = *cfg
		return
	}
}

func withDHTClient() Option {
	return func(r *Runtime) (err error) {
		r.buildCfg.Routing = libp2p.DHTClientOption
		return
	}
}

func withNilRepo() Option {
	return func(r *Runtime) (err error) {
		r.buildCfg.Repo = nil // for good measure
		r.buildCfg.NilRepo = true
		return
	}
}

func withDefault(opt []Option) []Option {
	return append([]Option{
		WithLogger(log.New(log.OptLevel(log.FatalLevel))),
		WithNamespace("ww"),
		withTTL(time.Second * 6),
		withBuildConfig(nil),
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

func defaultBuildConfig() *core.BuildCfg {
	// N.B.:  Repo will be set by `provideRepo`
	return &core.BuildCfg{
		Online:    true,
		Routing:   libp2p.DHTOption,
		Permanent: true,
		ExtraOpts: map[string]bool{
			"pubsub": true,
			// "ipnsps": false,
			// "mplex":  false,
		},
	}
}
