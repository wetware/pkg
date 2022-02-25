package runtime

import (
	"context"
	"net"
	"net/url"
	"strconv"
	"strings"

	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/metrics"
	disc "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"

	"github.com/lthibault/log"
	"github.com/thejerf/suture/v4"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"github.com/wetware/casm/pkg/boot"
	"github.com/wetware/casm/pkg/boot/crawl"
	bootutil "github.com/wetware/ww/internal/util/boot"
	statsdutil "github.com/wetware/ww/internal/util/statsd"
	"github.com/wetware/ww/pkg/vat"
)

var network = fx.Provide(
	bootstrap,
	vatNetwork,
	overlay,
	bootutil.NewCrawler,
	beacon)

type networkModule struct {
	fx.Out

	Vat vat.Network
	DHT *dual.DHT
}

func vatNetwork(c *cli.Context, lx fx.Lifecycle, b *metrics.BandwidthCounter) (mod networkModule, err error) {
	mod.Vat.NS = c.String("ns")
	mod.Vat.Host, err = libp2p.New(c.Context,
		libp2p.NoTransports,
		libp2p.Transport(libp2pquic.NewTransport),
		libp2p.ListenAddrStrings(c.StringSlice("listen")...),
		libp2p.BandwidthReporter(b))
	if err != nil {
		return
	}

	lx.Append(closer(mod.Vat.Host))

	dht, err := dual.New(c.Context, mod.Vat.Host,
		dual.LanDHTOption(dht.Mode(dht.ModeServer)),
		dual.WanDHTOption(dht.Mode(dht.ModeAuto)))
	if err != nil {
		return
	}

	lx.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return dht.Bootstrap(ctx)
		},
		OnStop: func(context.Context) error {
			return dht.Close()
		},
	})

	mod.Vat.Host = routedhost.Wrap(mod.Vat.Host, dht)
	return
}

func overlay(c *cli.Context, h host.Host, d discovery.Discovery) (*pubsub.PubSub, error) {
	return pubsub.NewGossipSub(c.Context, h,
		pubsub.WithRawTracer(statsdutil.NewPubSubTracer(c)),
		pubsub.WithDiscovery(d))
}

type bootstrapConfig struct {
	fx.In

	Logger    log.Logger
	Vat       vat.Network
	Datastore ds.Batching
	DHT       *dual.DHT

	Crawler    crawl.Crawler
	Beacon     crawl.Beacon
	Supervisor *suture.Supervisor

	Lifecycle fx.Lifecycle
}

func bootstrap(config bootstrapConfig) (discovery.Discovery, error) {
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
		Match:   pubsubTopic(config.Vat.NS),
		Target:  d,
		Default: disc.NewRoutingDiscovery(config.DHT),
	}, nil
}

func beacon(c *cli.Context, log log.Logger, vat vat.Network) (crawl.Beacon, error) {
	u, err := url.Parse(c.String("discover"))
	if err != nil {
		return crawl.Beacon{}, err
	}

	port, err := strconv.Atoi(u.Port())
	if err != nil {
		return crawl.Beacon{}, err
	}

	return crawl.Beacon{
		Logger: log.WithField("beacon_port", port),
		Addr:   &net.TCPAddr{Port: port},
		Host:   vat.Host,
	}, nil
}

func pubsubTopic(match string) func(string) bool {
	const prefix = "floodsub:"

	return func(s string) bool {
		return match == strings.TrimPrefix(s, prefix)
	}
}
