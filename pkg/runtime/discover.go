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
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/boot"
	"github.com/wetware/casm/pkg/boot/socket"
	bootutil "github.com/wetware/casm/pkg/boot/util"
	"github.com/wetware/casm/pkg/pex"
	"github.com/wetware/casm/pkg/util/metrics"
	"go.uber.org/fx"
)

func (c Config) ClientBootstrap() fx.Option {
	return fx.Provide(c.newClientDisc)
}

func (c Config) ServerBootstrap() fx.Option {
	return fx.Provide(c.newServerDisc)
}

func (c Config) newServerDisc(env Env, lx fx.Lifecycle, vat casm.Vat) (d discovery.Discovery, err error) {
	if env.IsSet("addr") {
		d, err = boot.NewStaticAddrStrings(env.StringSlice("addr")...)
		return
	}

	d, err = bootutil.ListenString(vat.Host, env.String("discover"),
		socket.WithLogger(env.Log()),
		socket.WithRateLimiter(socket.NewPacketLimiter(256, 16)))
	if c, ok := d.(io.Closer); ok {
		lx.Append(closer(c))
	}

	return
}

func (c Config) newClientDisc(env Env, lx fx.Lifecycle, vat casm.Vat) (d discovery.Discoverer, err error) {
	if env.IsSet("addr") {
		d, err = boot.NewStaticAddrStrings(env.StringSlice("addr")...)
		return
	}

	d, err = bootutil.DialString(vat.Host, env.String("discover"),
		socket.WithLogger(env.Log()),
		socket.WithRateLimiter(socket.NewPacketLimiter(256, 16)))
	if c, ok := d.(io.Closer); ok {
		lx.Append(closer(c))
	}

	return &logMetricDisc{
		disc:    d,
		metrics: bootMetrics{env},
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

	Env       Env
	Vat       casm.Vat
	DHT       *dual.DHT
	Datastore ds.Batching
	Lifecycle fx.Lifecycle
}

func (config psBootConfig) maybePeX(d discovery.Discovery, opt []pex.Option) (discovery.Discovery, error) {
	// pex disabled?
	if opt == nil {
		return d, nil
	}

	px, err := pex.New(config.Vat.Host, append([]pex.Option{
		// default options for PeX
		pex.WithLogger(config.Env.Log()),
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
	bootTopic := "floodsub:" + config.Env.String("ns")
	match := func(ns string) bool {
		return ns == bootTopic
	}

	target := logMetricDisc{
		disc:    d,
		advt:    d,
		metrics: bootMetrics{config.Env},
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
	env interface {
		Log() log.Logger
		Metrics() metrics.Client
	}
}

func (m bootMetrics) OnFindPeers(ns string) {
	m.env.Log().Debug("bootstrapping namespace")
	m.env.Metrics().Incr(fmt.Sprintf("boot.%s.find_peers", ns))
}

func (m bootMetrics) OnAdvertise(ns string) {
	m.env.Log().Debug("advertising namespace")
	m.env.Metrics().Decr(fmt.Sprintf("boot.%s.find_peers", ns))
}
