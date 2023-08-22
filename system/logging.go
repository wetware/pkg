package system

import (
	"context"
	"errors"
	"log/slog"

	"capnproto.org/go/capnp/v3/exc"
	"github.com/wetware/pkg/util/log"
)

// ErrorReporter handles Cap'n Proto RPC errors.
type ErrorReporter struct {
	log.Logger
}

func (r ErrorReporter) ReportError(err error) {
	if err == nil || errors.Is(err, context.Canceled) {
		return
	}

	if r.Logger == nil {
		r.Logger = slog.Default()
	}

	switch e := err.(type) {
	case *exc.Exception:
		switch e.Type {
		case exc.Disconnected, exc.Failed:
			r.Debug(e.Error(),
				"exception", e.Type)
			return

		case exc.Overloaded:
			r.Warn(e.Error(),
				"exception", e.Type)
			return

		case exc.Unimplemented:
			r.Error(e.Error(),
				"exception", e.Type)
			return
		}
		panic(e.Type)

	default:
		r.Debug(err.Error())
	}
}
