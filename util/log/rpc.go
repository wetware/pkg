package log

import (
	"fmt"

	"capnproto.org/go/capnp/v3/exc"
	"golang.org/x/exp/slog"
)

// ErrorReporter handles Cap'n Proto RPC errors.
type ErrorReporter struct {
	Logger
}

func (log ErrorReporter) ReportError(err error) {
	if log.Logger == nil {
		log.Logger = slog.Default()
	}

	switch e := err.(type) {
	case nil:
		// No error; nothing to report.

	case *exc.Exception:
		switch e.Type {
		case exc.Failed:
			log.Error(err.Error(),
				"exception", e.Type)

		case exc.Overloaded:
			log.Warn(err.Error(),
				"exception", e.Type)

		case exc.Disconnected:
			log.Debug(err.Error(),
				"exception", e.Type)

		case exc.Unimplemented:
			log.Warn(err.Error(),
				"exception", e.Type)

		default:
			log.Info(err.Error())
		}

	default:
		log.Error("panic",
			"reason", "unhandled error type",
			"error", err)
		panic(fmt.Errorf("unhandled error type: %w", err))
	}
}
