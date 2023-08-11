package system

import (
	"os"

	"golang.org/x/exp/slog"
)

func init() {
	h := slog.NewJSONHandler(os.Stderr, nil)
	root := slog.New(h).With("rom", os.Args[0])
	slog.SetDefault(root)
}

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
