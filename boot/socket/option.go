package socket

import (
	"net"

	"github.com/lthibault/log"
)

type Option func(*Socket)

// WithLogger sets the logger instance.
// If l == nil, a default logger is used.
func WithLogger(l log.Logger) Option {
	if l == nil {
		l = log.New()
	}

	return func(s *Socket) {
		s.log = l
	}
}

// WithValidator sets the socket's record validator.
// If v == nil, a default validator is used.
func WithValidator(v RecordValidator) Option {
	if v == nil {
		v = BasicValidator("")
	}

	return func(s *Socket) {
		s.conn.validate = v
	}
}

// WithErrHandler sets the socket's error callback.  If h == nil,
// a default error handler is used, which logs errors using the
// socket's logger.
func WithErrHandler(h func(*Socket, error)) Option {
	if h == nil {
		h = func(sock *Socket, err error) {
			select {
			case <-sock.Done(): // if sock is closed, log as debug
				sock.Log().Debug(err)
				return
			default:
			}

			switch e := err.(type) {
			case net.Error:
				sock.Log().Error(err)

			case ProtocolError:
				sock.Log().With(e).Debug(e.Message)

			default:
				sock.Log().WithError(err).Error("socket error")
			}
		}
	}

	return func(s *Socket) {
		s.handleError = h
	}
}

// WithCache sets the socket's record cache. If cache == nil,
// a default cache with 8 slots is used.
func WithCache(cache *RecordCache) Option {
	if cache == nil {
		cache = NewCache(8)
	}

	return func(s *Socket) {
		s.cache = cache
	}
}

// WithRateLimiter sets the socket's rate-limiter.  If
// lim == nil, a nop limiter is used.
func WithRateLimiter(lim *RateLimiter) Option {
	return func(s *Socket) {
		s.conn.lim = lim
	}
}

func withDefault(opt []Option) []Option {
	return append([]Option{
		WithCache(nil),
		WithLogger(nil),
		WithValidator(nil),
		WithErrHandler(nil),
	}, opt...)
}
