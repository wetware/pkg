package beacon

import (
	"context"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/wetware/ww/pkg/boot"
	"github.com/wetware/ww/pkg/runtime"
	"github.com/wetware/ww/pkg/runtime/svc/internal"
	"go.uber.org/fx"
)

// Config for Beacon service.
type Config struct {
	fx.In

	Host     host.Host
	Strategy boot.Strategy
}

// NewService satisfies runtime.ServiceFactory.
func (cfg Config) NewService() (runtime.Service, error) {
	if b, ok := cfg.Strategy.(boot.Beacon); ok {
		return beaconService{Beacon: b, h: cfg.Host}, nil
	}

	return nopService{}, nil

}

// Module for Beacon service.
type Module struct {
	fx.Out

	Factory runtime.ServiceFactory `group:"runtime"`
}

// New Beacon service.  Starts a local local server to respond to boot requests, if such
// a server is required by the boot strategy.
func New(cfg Config) Module { return Module{Factory: cfg} }

type beaconService struct {
	h host.Host
	boot.Beacon
}

func (b beaconService) Start(ctx context.Context) (err error) {
	if err = internal.WaitNetworkReady(ctx, b.h.EventBus()); err == nil {
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
