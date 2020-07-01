package runtimeutil

import (
	"context"

	"github.com/libp2p/go-libp2p-core/event"

	"github.com/lthibault/wetware/pkg/runtime"
)

// EventBusser provides access to an asynchronous event bus.
type EventBusser interface {
	EventBus() event.Bus
}

// CoreEventStream subscribes to core runtime events on b's event bus.
// It returns the channel supplied by the `Out()` method of the underlying
// `event.Subscription.Out`, which may contain any of the following events:
//
// - runtime.Exception
// - runtime.EvtServiceState
//
// It is closed when the context expires.
func CoreEventStream(ctx context.Context, b EventBusser) (out <-chan interface{}, err error) {
	var sub event.Subscription
	if sub, err = b.EventBus().Subscribe([]interface{}{
		new(runtime.Exception),
		new(runtime.EvtServiceStateChanged),
	}); err == nil {
		out = sub.Out()
		go func() {
			<-ctx.Done()
			sub.Close()
		}()
	}

	return
}
