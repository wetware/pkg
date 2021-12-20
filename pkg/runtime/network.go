package runtime

import (
	"context"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	disc "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"
	"github.com/thejerf/suture/v4"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"github.com/wetware/casm/pkg/boot"
	"github.com/wetware/casm/pkg/cluster"
	bootutil "github.com/wetware/ww/internal/util/boot"
)

var network = fx.Options(
	fx.Provide(
		bootstrap,
		routing,
		overlay,
		bootutil.NewCrawler,
		beacon,
		node),
	fx.Invoke(relayTopics))

type clusterConfig struct {
	fx.In

	Logger log.Logger
	PubSub *pubsub.PubSub

	Lifecycle fx.Lifecycle
}

func node(c *cli.Context, config clusterConfig) (*cluster.Node, error) {
	node, err := cluster.New(c.Context, config.PubSub,
		cluster.WithLogger(config.Logger),
		cluster.WithNamespace(c.String("ns")))

	if err == nil {
		config.Lifecycle.Append(closer(node))
	}

	return node, err
}

func routing(c *cli.Context, h host.Host) (*dual.DHT, error) {
	return dual.New(c.Context, h,
		dual.LanDHTOption(dht.Mode(dht.ModeServer)),
		dual.WanDHTOption(dht.Mode(dht.ModeAuto)))
}

func overlay(c *cli.Context, h host.Host, d discovery.Discovery) (*pubsub.PubSub, error) {
	return pubsub.NewGossipSub(c.Context, h, pubsub.WithDiscovery(d))
}

type bootstrapConfig struct {
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

func bootstrap(c *cli.Context, config bootstrapConfig) (discovery.Discovery, error) {
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

	d := struct {
		discovery.Discoverer
		discovery.Advertiser
	}{
		Discoverer: config.Crawler,
		Advertiser: config.Beacon,
	}

	// TODO:  enable PeX when testing is complete

	// // Wrap the bootstrap discovery service in a peer sampling service.
	// px, err := pex.New(c.Context, config.Host,
	// 	pex.WithLogger(config.Logger),
	// 	pex.WithDatastore(config.Datastore),
	// 	pex.WithDiscovery(d))
	// if err != nil {
	// 	return nil, err
	// }

	// If the namespace matches the cluster pubsub topic,
	// fetch peers from PeX, which itself will fall back
	// on the bootstrap services.
	return boot.Namespace{
		Match:   pubsubTopic(c.String("ns")),
		Target:  d,
		Default: disc.NewRoutingDiscovery(config.DHT),
	}, nil
}

func beacon(c *cli.Context, log log.Logger, h host.Host) (boot.Beacon, error) {
	u, err := url.Parse(c.String("discover"))
	if err != nil {
		return boot.Beacon{}, err
	}

	port, err := strconv.Atoi(u.Port())
	if err != nil {
		return boot.Beacon{}, err
	}

	return boot.Beacon{
		Logger: log.WithField("beacon_port", port),
		Addr:   &net.TCPAddr{Port: port},
		Host:   h,
	}, nil
}

func pubsubTopic(match string) func(string) bool {
	const prefix = "floodsub:"

	return func(s string) bool {
		return match == strings.TrimPrefix(s, prefix)
	}
}

func timeout(ctx context.Context) time.Duration {
	if t, ok := ctx.Deadline(); ok {
		return time.Until(t)
	}

	return time.Second * 5
}

func relayTopics(c *cli.Context, log log.Logger, ps *pubsub.PubSub, lx fx.Lifecycle) {
	for _, topic := range c.StringSlice("relay") {
		lx.Append(newRelayHook(log.WithField("topic", topic), ps, topic))
	}
}

func newRelayHook(log log.Logger, ps *pubsub.PubSub, topic string) fx.Hook {
	var (
		t      *pubsub.Topic
		cancel pubsub.RelayCancelFunc
	)

	return fx.Hook{
		OnStart: func(context.Context) (err error) {
			if t, err = ps.Join(topic); err != nil {
				return
			}

			if cancel, err = t.Relay(); err != nil {
				return
			}

			log.Info("relaying topic")
			return
		},
		OnStop: func(ctx context.Context) error {
			cancel()
			return t.Close()
		},
	}
}
