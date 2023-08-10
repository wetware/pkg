package socket

import (
	"net"

	"golang.org/x/exp/slog"
)

// Logger is used for logging by the RPC system. Each method logs
// messages at a different level, but otherwise has the same semantics:
//
//   - Message is a human-readable description of the log event.
//   - Args is a sequenece of key, value pairs, where the keys must be strings
//     and the values may be any type.
//   - The methods may not block for long periods of time.
//
// This interface is designed such that it is satisfied by *slog.Logger.
type Logger interface {
	Debug(message string, args ...any)
	Info(message string, args ...any)
	Warn(message string, args ...any)
	Error(message string, args ...any)
}

type Option func(*Socket)

// WithLogger sets the logger instance.
// If l == nil, a default logger is used.
func WithLogger(l Logger) Option {
	if l == nil {
		l = slog.Default()
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
			// socket already closed?
			case <-sock.Done():
				sock.Log().Debug("got error from after closing socket",
					"error", err.Error())
				return
			default:
			}

			switch e := err.(type) {
			case net.Error:
				sock.Log().Error(err.Error(),
					"timeout", e.Timeout())

			case ProtocolError:
				sock.Log().Debug(e.Message,
					"cause", e.Cause)

			default:
				sock.Log().Error("socket error",
					"error", e.Error())
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
