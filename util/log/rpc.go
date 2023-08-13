package log

import (
	"errors"
	"io"

	"capnproto.org/go/capnp/v3/exc"
	"golang.org/x/exp/slog"
)

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
