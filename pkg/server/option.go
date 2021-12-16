package server

import (
	"github.com/lthibault/log"
)

// Option type for Node
type Option func(*Node)

func WithLogger(l log.Logger) Option {
	if l == nil {
		l = log.New()
	}

	return func(n *Node) {
		n.log = l
	}
}

func WithNamespace(ns string) Option {
	if ns == "" {
		ns = "ww"
	}

	return func(n *Node) {
		n.ns = ns
	}
}

func withDefaults(opt []Option) []Option {
	return append([]Option{
		WithLogger(nil),
	}, opt...)
}
