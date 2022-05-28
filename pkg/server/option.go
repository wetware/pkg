package server

import (
	"github.com/lthibault/log"
	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/ww/pkg/cap/anchor"
)

type Option func(*Joiner)

// WithLogger sets the logger for the peer exchange.
// If l == nil, a default logger is used.
func WithLogger(l log.Logger) Option {
	if l == nil {
		l = log.New()
	}

	return func(j *Joiner) {
		j.log = l
	}
}

// WithMerge specifies how the host node should merge clusters
// during Join calls. If m == nil, a default strategy is used,
// which simply connects to the remote vat.
func WithMerge(m anchor.MergeStrategy) Option {
	f := newMergeFactory(m)
	return func(j *Joiner) {
		j.newMerge = f
	}
}

func WithClusterConfig(opt ...cluster.Option) Option {
	return func(j *Joiner) {
		j.opts = opt
	}
}

func withDefault(opt []Option) []Option {
	return append([]Option{
		WithLogger(nil),
	}, opt...)
}
