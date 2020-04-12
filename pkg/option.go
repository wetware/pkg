package ww

import log "github.com/lthibault/log/pkg"

// Option type for Host
type Option func(*Runtime) error

// WithLogger sets the logger.
func WithLogger(logger log.Logger) Option {
	return func(r *Runtime) (err error) {
		r.log = logger
		return
	}
}

func withDefault(opt []Option) []Option {
	return append([]Option{
		WithLogger(log.New(log.OptLevel(log.FatalLevel))),
	}, opt...)
}
