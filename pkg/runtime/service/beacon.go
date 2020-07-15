package service

import (
	"context"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/lthibault/wetware/pkg/boot"
	"github.com/lthibault/wetware/pkg/runtime"
)

// Beacon starts a local local server to respond to boot requests, if such a server
// is required by the boot strategy.
func Beacon(h host.Host, b boot.Strategy) ProviderFunc {
	return func() (runtime.Service, error) {
		if bc, ok := b.(boot.Beacon); ok {
			return beaconService{Beacon: bc, h: h}, nil
		}

		return nopService{}, nil
	}
}

type beaconService struct {
	h host.Host
	boot.Beacon
}

func (b beaconService) Start(ctx context.Context) (err error) {
	if err = waitNetworkReady(ctx, b.h.EventBus()); err == nil {
		err = b.Beacon.Signal(ctx, b.h)
	}

	return
}

func (b beaconService) Stop(ctx context.Context) error {
	return b.Beacon.Stop(ctx)
}

type nopService struct{}

func (nopService) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"service": "nop",
	}
}

func (nopService) Start(context.Context) error {
	return nil
}

func (nopService) Stop(context.Context) error {
	return nil
}
