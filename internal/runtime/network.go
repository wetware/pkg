package runtime

import (
	"context"
	"net"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	disc "github.com/libp2p/go-libp2p-discovery"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"
	"github.com/thejerf/suture/v4"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"github.com/wetware/casm/pkg/boot"
	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/casm/pkg/pex"
)

func bindNetwork() fx.Option {
	return fx.Provide(
		bindCluster,
		bindPubSub,
		bindDiscovery,
		bindCrawler)
}

type clusterConfig struct {
	fx.In

	Logger log.Logger
	PubSub *pubsub.PubSub

	Lifecycle fx.Lifecycle
}

func bindCluster(c *cli.Context, config clusterConfig) (*cluster.Node, error) {
	node, err := cluster.New(c.Context, config.PubSub,
		cluster.WithLogger(config.Logger),
		cluster.WithNamespace(c.String("ns")))

	if err == nil {
		config.Lifecycle.Append(closer(node))
	}

	return node, err
}

func bindPubSub(c *cli.Context, h host.Host, d discovery.Discovery) (*pubsub.PubSub, error) {
	return pubsub.NewGossipSub(c.Context, h, pubsub.WithDiscovery(d))
}

type discoveryConfig struct {
	fx.In

	Logger    log.Logger
	Host      host.Host
	Datastore ds.Batching
	DHT       *dual.DHT

	Crawler    boot.Crawler
	Beacon     boot.Beacon
	Supervisor *suture.Supervisor

	Lifecycle fx.Lifecycle
}

func bindDiscovery(c *cli.Context, config discoveryConfig) (discovery.Discovery, error) {
	var token suture.ServiceToken
	config.Lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			token = config.Supervisor.Add(config.Beacon)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return config.Supervisor.RemoveAndWait(token, timeout(ctx))
		},
	})

	// Wrap the bootstrap discovery service in a peer sampling service.
	px, err := pex.New(c.Context, config.Host,
		pex.WithLogger(config.Logger),
		pex.WithDatastore(config.Datastore),
		pex.WithDiscovery(struct {
			discovery.Discoverer
			discovery.Advertiser
		}{
			Discoverer: config.Crawler,
			Advertiser: config.Beacon,
		}))

	// If the namespace matches the cluster pubsub topic,
	// fetch peers from PeX, which itself will fall back
	// on the bootstrap service 'p'.
	return boot.Cache{
		Match: exactly(c.String("ns")),
		Cache: px,
		Else:  disc.NewRoutingDiscovery(config.DHT),
	}, err
}

func exactly(match string) func(string) bool {
	return func(s string) bool {
		return match == s
	}
}

func bindCrawler(c *cli.Context, log log.Logger) boot.Crawler {
	return boot.Crawler{
		Dialer: new(net.Dialer),
		Strategy: &boot.ScanSubnet{
			Logger: log,
			Net:    "tcp",
			Port:   8822,
			CIDR:   "127.0.0.1/24", // XXX
		},
	}
}

func timeout(ctx context.Context) time.Duration {
	if t, ok := ctx.Deadline(); ok {
		return time.Until(t)
	}

	return time.Second * 5
}
