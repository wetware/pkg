package p2p

import (
	"context"

	"go.uber.org/fx"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	discovery "github.com/libp2p/go-libp2p-discovery"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/config"
)

// Config for p2p layer
type Config struct {
	fx.In

	HostOpt []config.Option
	DHTOpt  []dual.Option
}

// Module encapsulates p2p primitives
type Module struct {
	fx.Out

	DHT       routing.Routing
	Host      host.Host
	EventBus  event.Bus
	PubSub    *pubsub.PubSub
	Discovery discovery.Discovery
}

// New p2p module.
func New(ctx context.Context, cfg Config, lx fx.Lifecycle) (mod Module, err error) {
	if mod.Host, err = newHost(ctx, lx, cfg.HostOpt...); err != nil {
		return
	}

	if mod.DHT, err = dual.New(ctx, mod.Host, cfg.DHTOpt...); err != nil {
		return
	}

	if mod.Host, err = wrapHost(mod.Host, mod.DHT); err != nil {
		return
	}

	mod.EventBus = mod.Host.EventBus()
	mod.Discovery = discovery.NewRoutingDiscovery(mod.DHT)

	if mod.PubSub, err = pubsub.NewGossipSub(
		ctx,
		mod.Host,
		pubsub.WithDiscovery(mod.Discovery),
	); err != nil {
		return
	}

	return
}

func newHost(ctx context.Context, lx fx.Lifecycle, opt ...config.Option) (h host.Host, err error) {
	if h, err = libp2p.New(ctx, opt...); err == nil {
		lx.Append(fx.Hook{
			OnStop: func(context.Context) error {
				return h.Close()
			},
		})
	}
	return
}
