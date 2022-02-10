package runtime

import (
	"context"
	"net"
	"net/url"
	"strconv"
	"strings"

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
	bootutil "github.com/wetware/ww/internal/util/boot"
	statsdutil "github.com/wetware/ww/internal/util/statsd"
)

var network = fx.Provide(
	bootstrap,
	routing,
	overlay,
	bootutil.NewCrawler,
	beacon)

func routing(c *cli.Context, h host.Host, lx fx.Lifecycle) (*dual.DHT, error) {
	dht, err := dual.New(c.Context, h,
		dual.LanDHTOption(dht.Mode(dht.ModeServer)),
		dual.WanDHTOption(dht.Mode(dht.ModeAuto)))

	if err == nil {
		lx.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				return dht.Bootstrap(ctx)
			},
			OnStop: func(context.Context) error {
				return dht.Close()
			},
		})
	}

	return dht, err
}

func overlay(c *cli.Context, h host.Host, d discovery.Discovery) (*pubsub.PubSub, error) {
	return pubsub.NewGossipSub(c.Context, h,
		pubsub.WithRawTracer(statsdutil.NewPubSubTracer(c)),
		pubsub.WithDiscovery(d))
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
	config.Lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			config.Supervisor.Add(config.Beacon)
			return nil
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
