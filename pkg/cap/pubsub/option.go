package pubsub

import (
	"github.com/lthibault/log"
)

type Option func(*Factory)

func WithLogger(l log.Logger) Option {
	if l == nil {
		l = log.New()
	}

	return func(f *Factory) {
		f.log = l
	}
}

func withDefault(opt []Option) []Option {
	return append([]Option{
		WithLogger(nil),
	}, opt...)
}
