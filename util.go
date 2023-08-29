package ww

import (
	"context"
	"log/slog"

	"github.com/thejerf/suture/v4"
)

func (vat Vat) ListenAndServe(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	app := suture.New(vat.String(), suture.Spec{
		EventHook: vat.OnEvent,
	})
	app.Add(vat.Server)

	return app.Serve(ctx)
}

func (vat Vat) OnEvent(event suture.Event) {
	switch e := event.(type) {

	case suture.EventStopTimeout:
		slog.Error("shutdown failed",
			"event", e)

	case suture.EventServicePanic:
		slog.Error("crashed",
			"event", e)

	case suture.EventServiceTerminate:
		slog.Warn("terminated",
			"event", e)

	case *suture.EventBackoff:
		slog.Info("paused",
			"event", e)

	case suture.EventResume:
		slog.Info("resumed",
			"event", e)

	}
}
