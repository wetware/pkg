// Package embed provides utilities for embedding ww server nodes into applications.
package embed

import (
	"github.com/libp2p/go-libp2p-core/pnet"
	"github.com/lthibault/log"
	"go.uber.org/fx"

	"github.com/wetware/ww/pkg/server"
)

// ServerConfig can be populated by Fx.
type ServerConfig struct {
	fx.In

	Logger  log.Logger
	Topics  []string                 `group:"topics"`
	Host    server.RoutedHostFactory `optional:"true"`
	DHT     server.DualDHTFactory    `optional:"true"`
	PubSub  server.GossipsubFactory  `optional:"true"`
	Secret  pnet.PSK                 `optional:"true"`
	Cluster server.ClusterConfig     `optional:"true"`
}

// Server returns a fully configured 'server.Node', suitable for
// embedding in applications. The contents of 'cfg' is populated
// by Fx if 'Server' is provided as a dependency.
func Server(cfg ServerConfig) server.Node {
	return server.New(
		server.WithLogger(cfg.Logger),
		server.WithTopics(cfg.Topics...),
		server.WithHost(&cfg.Host),
		server.WithDHT(&cfg.DHT),
		server.WithPubSub(&cfg.PubSub),
		server.WithSecret(cfg.Secret),
		server.WithClusterConfig(cfg.Cluster))
}
