package runtime

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/lthibault/log"

	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"go.uber.org/fx"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/pex"
	"github.com/wetware/casm/pkg/util/metrics"
	"github.com/wetware/ww/boot"
	"github.com/wetware/ww/boot/socket"
)

type bootConfig struct {
	fx.In

	Log     log.Logger
	Metrics metrics.Client
	Vat     casm.Vat
	Flag    Flags
}

func (bc bootConfig) host() host.Host {
	return bc.Vat.Host
}

func (bc bootConfig) metrics() bootMetrics {
	return bootMetrics{
		Log:     bc.Log,
		Metrics: bc.Metrics,
	}
}

func (c Config) ClientBootstrap() fx.Option {
	return fx.Provide(c.newClientDisc)
}

func (c Config) ServerBootstrap() fx.Option {
	return fx.Provide(c.newServerDisc)
}

func (c Config) newServerDisc(config bootConfig, lx fx.Lifecycle) (d discovery.Discovery, err error) {
	if config.Flag.IsSet("addr") {
		d, err = boot.NewStaticAddrStrings(config.Flag.StringSlice("addr")...)
		return
	}

	d, err = boot.ListenString(config.host(), config.Flag.String("discover"),
		socket.WithLogger(config.Log),
		socket.WithRateLimiter(socket.NewPacketLimiter(256, 16)))
	if c, ok := d.(io.Closer); ok {
		lx.Append(closer(c))
	}

	return
}

func (c Config) newClientDisc(config bootConfig, lx fx.Lifecycle) (d discovery.Discoverer, err error) {
	if config.Flag.IsSet("addr") {
		d, err = boot.NewStaticAddrStrings(config.Flag.StringSlice("addr")...)
		return
	}

	d, err = boot.DialString(config.host(), config.Flag.String("discover"),
		socket.WithLogger(config.Log),
		socket.WithRateLimiter(socket.NewPacketLimiter(256, 16)))
	if c, ok := d.(io.Closer); ok {
		lx.Append(closer(c))
	}

	return &logMetricDisc{
		disc:    d,
		metrics: config.metrics(),
	}, err
}

func (c Config) withPubSubDiscovery(d discovery.Discovery, config psBootConfig) (discovery.Discovery, error) {
	d, err := config.maybePeX(d, c.pexOpt)
	if err == nil {
		d = config.Wrap(d)
	}

	return d, err
}

type psBootConfig struct {
	fx.In

	Boot      bootConfig
	DHT       *dual.DHT
	Datastore ds.Batching
	Lifecycle fx.Lifecycle
}

func (config psBootConfig) maybePeX(d discovery.Discovery, opt []pex.Option) (discovery.Discovery, error) {
	// pex disabled?
	if opt == nil {
		return d, nil
	}

	px, err := pex.New(config.Boot.host(), append([]pex.Option{
		// default options for PeX
		pex.WithLogger(config.Boot.Log),
		pex.WithDatastore(config.Datastore),
		pex.WithDiscovery(d),
	}, opt...)...)

	if err == nil {
		config.Lifecycle.Append(closer(px))
	}

	return px, err
}

func (config psBootConfig) Wrap(d discovery.Discovery) *boot.Namespace {
	// Dynamically dispatch advertisements and queries to either:
	//
	//  1. the bootstrap service, iff namespace matches cluster topic; else
	//  2. the DHT-backed discovery service.
	bootTopic := "floodsub:" + config.Boot.Flag.String("ns")
	match := func(ns string) bool {
		return ns == bootTopic
	}

	target := logMetricDisc{
		disc:    d,
		advt:    d,
		metrics: config.Boot.metrics(),
	}

	return &boot.Namespace{
		Match:   match,
		Target:  trimPrefixDisc{target},
		Default: routing.NewRoutingDiscovery(config.DHT),
	}
}

// Trims the "floodsub:" prefix from the namespace.  This is needed because
// clients do not use pubsub, and will search for the exact namespace string.
type trimPrefixDisc struct{ discovery.Discovery }

func (b trimPrefixDisc) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	ns = strings.TrimPrefix(ns, "floodsub:")
	return b.Discovery.FindPeers(ctx, ns, opt...)
}

func (b trimPrefixDisc) Advertise(ctx context.Context, ns string, opt ...discovery.Option) (time.Duration, error) {
	ns = strings.TrimPrefix(ns, "floodsub:")
	return b.Discovery.Advertise(ctx, ns, opt...)
}

type logMetricDisc struct {
	metrics bootMetrics
	disc    discovery.Discoverer
	advt    discovery.Advertiser
}

func (b logMetricDisc) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	b.metrics.OnFindPeers(ns)
	return b.disc.FindPeers(ctx, ns, opt...)
}

func (b logMetricDisc) Advertise(ctx context.Context, ns string, opt ...discovery.Option) (time.Duration, error) {
	b.metrics.OnAdvertise(ns)
	return b.advt.Advertise(ctx, ns, opt...)
}

type bootMetrics struct {
	Log     log.Logger
	Metrics metrics.Client
}

func (m bootMetrics) OnFindPeers(ns string) {
	m.Log.Debug("bootstrapping namespace")
	m.Metrics.Incr(fmt.Sprintf("boot.%s.find_peers", ns))
}

func (m bootMetrics) OnAdvertise(ns string) {
	m.Log.Debug("advertising namespace")
	m.Metrics.Decr(fmt.Sprintf("boot.%s.find_peers", ns))
}
