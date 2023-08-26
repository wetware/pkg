package ww

import (
	"context"
	"log/slog"

	"github.com/thejerf/suture/v4"

	"capnproto.org/go/capnp/v3/rpc"
)

func (vat Vat[T]) ListenAndServe(ctx context.Context) error {
	vat.in = make(chan *rpc.Conn)

	app := suture.New(vat.String(), suture.Spec{
		EventHook: vat.OnEvent,
	})
	app.Add(vat)

	return app.Serve(ctx)
}

func (vat Vat[T]) OnEvent(event suture.Event) {
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
