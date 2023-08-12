//go:generate mockgen -source=log.go -destination=test/log.go -package=test_log

package log

import (
	"errors"
	"io"

	"capnproto.org/go/capnp/v3/exc"
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

// ErrorReporter handles Cap'n Proto RPC errors.
type ErrorReporter struct{ Logger }

func (log ErrorReporter) ReportError(err error) {
	if err != nil {

		if log.Logger == nil {
			log.Logger = slog.Default()
		}

		switch t := exc.TypeOf(err); t {
		case exc.Failed:
			if errors.Is(err, io.EOF) {
				return
			}

			log.Error(err.Error(),
				"exception", t)

		case exc.Overloaded:
			log.Warn(err.Error(),
				"exception", t)

		case exc.Disconnected:
			log.Debug(err.Error(),
				"exception", t)

		case exc.Unimplemented:
			log.Warn(err.Error(),
				"exception", t)

		default:
			log.Info(err.Error())
		}
	}
}
