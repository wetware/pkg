package server

import (
	"github.com/lthibault/log"
	"github.com/wetware/casm/pkg/cluster"
	statsdutil "github.com/wetware/ww/internal/util/statsd"
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

// WithStatsd sets the statsd client for recording statistics.
func WithMetrics(m *statsdutil.WwMetricsRecorder) Option {
	return func(j *Joiner) {
		j.metrics = m
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
