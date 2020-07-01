package service

import (
	"context"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/lthibault/wetware/pkg/internal/p2p"
	"github.com/lthibault/wetware/pkg/runtime"
	"github.com/pkg/errors"
)

// ProviderFunc satisfies runtime.ServiceFactory.
type ProviderFunc func() (runtime.Service, error)

// Service initializes a new runtime service.
func (f ProviderFunc) Service() (runtime.Service, error) {
	return f()
}

func netReadySubscription(bus event.Bus) (event.Subscription, error) {
	return bus.Subscribe(new(p2p.EvtNetworkReady))
}

func waitNetworkReady(ctx context.Context, bus event.Bus) error {
	sub, err := netReadySubscription(bus)
	if err != nil {
		return err
	}
	defer sub.Close()

	select {
	case <-sub.Out():
		return nil
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "wait network ready")
	}
}
