package pubsub

import (
	"github.com/lthibault/log"
)

type Option func(*Provider)

// WithLogger sets the logger for the peer exchange.
// If l == nil, a default logger is used.
func WithLogger(l log.Logger) Option {
	if l == nil {
		l = log.New()
	}

	return func(p *Provider) {
		p.log = l
	}
}

func withDefault(opt []Option) []Option {
	return append([]Option{
		WithLogger(nil),
	}, opt...)
}
