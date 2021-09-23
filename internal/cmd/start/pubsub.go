package start

import (
	"context"

	"github.com/libp2p/go-libp2p-core/host"
	disc "github.com/libp2p/go-libp2p-discovery"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"go.uber.org/fx"

	ww "github.com/wetware/ww/pkg"
)

type pubSubConfig struct {
	fx.In

	Ctx  context.Context
	Host host.Host
	DHT  ww.DHT
}

func newPubSub(cfg pubSubConfig, lx fx.Lifecycle) (ww.PubSub, error) {
	// TODO(enhancement):  PeX-based discovery
	return pubsub.NewGossipSub(
		cfg.Ctx,
		cfg.Host,
		pubsub.WithDiscovery(disc.NewRoutingDiscovery(cfg.DHT)))
}
