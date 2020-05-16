package eventloop

import (
	"context"

	"github.com/libp2p/go-libp2p-core/event"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"
)

// Handler .
type Handler struct {
	Type     interface{}
	Callback func(interface{})
	Opt      []event.SubscriptionOpt

	sub event.Subscription
}

func (h *Handler) init(bus event.Bus) (err error) {
	h.sub, err = bus.Subscribe(h.Type, h.Opt...)
	return
}

func (h *Handler) run() {
	for event := range h.sub.Out() {
		h.Callback(event)
	}
}

// RegisterHandlers for background execution.
func RegisterHandlers(lx fx.Lifecycle, bus event.Bus, hs ...Handler) (err error) {
	for _, h := range hs {
		if err = h.init(bus); err != nil {
			return
		}
	}

	// looping a second time guarantees we don't orphan goroutines
	// if any calls to `init` fail.
	for _, h := range hs {
		go h.run()
	}

	lx.Append(fx.Hook{
		OnStop: func(context.Context) error {
			var g errgroup.Group
			for _, h := range hs {
				g.Go(h.sub.Close)
			}
			return g.Wait()
		},
	})

	return
}
