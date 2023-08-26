package ww

import (
	"context"

	"capnproto.org/go/capnp/v3/rpc"

	"github.com/thejerf/suture/v4"
)

func (vat Vat[T]) ListenAndServe(ctx context.Context) error {
	vat.in = make(chan *rpc.Conn)

	app := suture.New(vat.String(), suture.Spec{
		// EventHook: vat.OnEvent,
	})
	app.Add(vat)

	return app.Serve(ctx)
}
