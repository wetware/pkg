package server

import (
	"github.com/lthibault/log"
	"github.com/wetware/casm/pkg/cluster"
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

func WithClusterOpts(opt ...cluster.Option) Option {
	opt = append([]cluster.Option{
		cluster.WithNamespace("ww"),
	}, opt...)

	return func(n *Node) {
		n.clusterOpt = opt
	}
}

func withDefault(opt []Option) []Option {
	return append([]Option{
		WithLogger(nil),
		WithClusterOpts(),
	}, opt...)
}
