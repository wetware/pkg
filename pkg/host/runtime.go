package host

import (
	"context"
	"time"

	"go.uber.org/fx"

	// libp2p
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/config"

	// libp2p core interfaces
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/pnet"

	// libp2p core implementations
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-peerstore/pstoreds"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	// libp2p/IPFS misc.
	"github.com/ipfs/go-datastore"
	"github.com/multiformats/go-multiaddr"

	// wetware utils
	ctxutil "github.com/wetware/ww/internal/util/ctx"
	hostutil "github.com/wetware/ww/internal/util/host"

	// wetware internal

	"github.com/wetware/ww/pkg/internal/filter"
	"github.com/wetware/ww/pkg/internal/p2p"
	"github.com/wetware/ww/pkg/lang"

	// wetware public
	"github.com/wetware/ww/pkg/boot"
	"github.com/wetware/ww/pkg/runtime"
	"github.com/wetware/ww/pkg/runtime/service"
)

const timestep = time.Millisecond * 100

func services(cfg serviceConfig) runtime.ServiceBundle {
	return runtime.Bundle(
		service.Ticker(cfg.Host.EventBus(), timestep),
		service.Filter(cfg.Host.EventBus(), cfg.RoutingTopic, cfg.Filter),
		service.ConnTracker(cfg.Host),
		service.Neighborhood(cfg.Host.EventBus(), cfg.Graph.KMin, cfg.Graph.KMax),
		service.Bootstrap(cfg.Host, cfg.Boot),
		service.Beacon(cfg.Host, cfg.Boot),
		// service.Discover(cfg.Host, cfg.Namespace, cfg.Discovery),
		service.Graph(cfg.Host),
		service.Announcer(cfg.Host, cfg.RoutingTopic, cfg.TTL),
		service.Joiner(cfg.Host),
	)
}

// Config for the server runtime.
type Config struct {
	ns         string
	ttl        time.Duration
	kmin, kmax int

	psk   pnet.PSK
	addrs []multiaddr.Multiaddr
	ds    datastore.Batching
	boot  boot.Strategy
}

func (cfg Config) assemble(h *Host) *fx.App {
	return fx.New(
		fx.NopLogger,
		fx.Populate(h),
		fx.Provide(
			cfg.options,
			p2p.New,
			routingTopic,
			// block.New,
			filter.New,
			services,
			newAnchor,
			lang.New,
			newHost,
		),
		fx.Invoke(
			runtime.Register,
			listen,
		),
	)
}

func (cfg Config) options(lx fx.Lifecycle) (mod module, err error) {
	mod.Ctx = ctxutil.WithLifecycle(context.Background(), lx) // libp2p lifecycle
	mod.Namespace = cfg.ns
	mod.TTL = cfg.ttl
	mod.Boot = cfg.boot
	mod.ListenAddrs = cfg.addrs
	mod.Graph = graphParams(cfg.kmin, cfg.kmax)

	var ps peerstore.Peerstore
	if ps, err = pstoreds.NewPeerstore(mod.Ctx, cfg.ds, pstoreds.DefaultOpts()); err != nil {
		return
	}

	cm := connmgr.NewConnManager(cfg.kmin, cfg.kmax, time.Second*10)

	mod.HostOpt = []config.Option{
		libp2p.DisableRelay(),
		hostutil.MaybePrivate(cfg.psk),
		libp2p.NoListenAddrs, // defer listening until setup is complete
		libp2p.UserAgent("ww-host"),
		libp2p.Peerstore(ps),
		libp2p.ConnectionManager(cm),
	}

	mod.DHTOpt = append(mod.DHTOpt, dual.DHTOption(
		dht.Datastore(cfg.ds),
		dht.Mode(dht.ModeServer),
	))

	return
}

type module struct {
	fx.Out

	Ctx       context.Context
	Namespace string        `name:"ns"`
	TTL       time.Duration `name:"ttl"`

	Graph struct{ KMin, KMax int }

	ListenAddrs []multiaddr.Multiaddr
	Boot        boot.Strategy

	HostOpt []config.Option
	DHTOpt  []dual.Option

	Datastore datastore.Batching
}

type serviceConfig struct {
	fx.In

	Namespace    string `name:"ns"`
	Graph        struct{ KMin, KMax int }
	Host         host.Host
	Boot         boot.Strategy
	Filter       filter.Filter
	RoutingTopic *pubsub.Topic
	TTL          time.Duration `name:"ttl"`
	Discovery    discovery.Discovery
}

func graphParams(kmin, kmax int) (ps struct{ KMin, KMax int }) {
	ps.KMin = kmin
	ps.KMax = kmax
	return
}

func listen(ctx context.Context, h host.Host, as []multiaddr.Multiaddr) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	return h.(p2p.Listener).Listen(ctx, as...)
}
