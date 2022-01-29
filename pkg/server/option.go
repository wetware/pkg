package server

import (
	"github.com/lthibault/log"
	"github.com/wetware/casm/pkg/cluster"
)

type Option func(*Joiner)

func WithLogger(l log.Logger) Option {
	if l == nil {
		l = log.New()
	}

	return func(j *Joiner) {
		j.log = l
	}
}

func WithNamespace(ns string) Option {
	if ns == "" {
		ns = "ww"
	}

	return func(j *Joiner) {
		j.ns = ns
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
		WithNamespace(""),
	}, opt...)
}
