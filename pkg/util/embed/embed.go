// Package embed provides utilities for embedding ww server nodes into applications.
package embed

import (
	"context"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/pnet"
	"github.com/lthibault/log"
	"go.uber.org/fx"

	"github.com/wetware/ww/pkg/cap"
	"github.com/wetware/ww/pkg/client"
	"github.com/wetware/ww/pkg/server"
)

// ServerConfig can be populated by Fx.
type ServerConfig struct {
	fx.In

	Logger  log.Logger
	Topics  []string             `group:"topics"`
	NS      string               `optional:"true"`
	Host    server.HostFactory   `optional:"true"`
	DHT     server.DHTFactory    `optional:"true"`
	PubSub  server.PubSubFactory `optional:"true"`
	Secret  pnet.PSK             `optional:"true"`
	Cluster server.ClusterConfig `optional:"true"`
}

// Server returns a fully configured 'server.Node', suitable for
// embedding in applications. The contents of 'cfg' is populated
// by Fx.
func Server(cfg ServerConfig) server.Node {
	return server.New(
		server.WithLogger(cfg.Logger),
		server.WithTopics(cfg.Topics...),
		server.WithHost(cfg.Host),
		server.WithDHT(cfg.DHT),
		server.WithPubSub(cfg.PubSub),
		server.WithSecret(cfg.Secret),
		server.WithClusterConfig(cfg.Cluster),
		server.WithNamepace(cfg.NS))
}

// DialConfig can be populated by Fx.
type DialConfig struct {
	fx.In

	Logger  log.Logger
	Join    discovery.Discoverer
	NS      string                `optional:"true" name:"ns"`
	Host    client.HostFactory    `optional:"true"`
	Routing client.RoutingFactory `optional:"true"`
	PubSub  client.PubSubFactory  `optional:"true"`
	Cap     cap.Dialer            `optional:"true"`
}

// Dialer returns a fully configured 'ClientDialer', suitable for
// embedding in applications.  The contents of 'cfg' is populated
// by Fx.
func Dialer(cfg DialConfig) ClientDialer { return ClientDialer(cfg) }

type ClientDialer DialConfig

func (d ClientDialer) Dial(ctx context.Context) (client.Node, error) {
	return client.DialDiscover(ctx, d.Join,
		client.WithLogger(d.Logger.WithField("ns", d.NS)),
		client.WithHost(d.Host),
		client.WithPubSub(d.PubSub),
		client.WithCapability(d.Cap),
		client.WithRouting(d.Routing),
		client.WithNamespace(d.NS))
}
