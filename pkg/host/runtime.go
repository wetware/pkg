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

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/pnet"

	// libp2p core implementations
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-peerstore/pstoreds"

	// libp2p/IPFS misc.
	"github.com/ipfs/go-datastore"
	"github.com/multiformats/go-multiaddr"

	// wetware utils
	ctxutil "github.com/wetware/ww/internal/util/ctx"
	hostutil "github.com/wetware/ww/internal/util/host"
	ww "github.com/wetware/ww/pkg"

	// wetware internal deps
	"github.com/wetware/ww/pkg/internal/p2p"

	// wetware public APIs
	"github.com/wetware/ww/pkg/boot"
	"github.com/wetware/ww/pkg/cluster"
	"github.com/wetware/ww/pkg/runtime"

	// runtime services
	announcer_service "github.com/wetware/ww/pkg/runtime/svc/announcer"
	beacon_service "github.com/wetware/ww/pkg/runtime/svc/beacon"
	boot_service "github.com/wetware/ww/pkg/runtime/svc/boot"
	epoch_service "github.com/wetware/ww/pkg/runtime/svc/epoch"
	graph_service "github.com/wetware/ww/pkg/runtime/svc/graph"
	join_service "github.com/wetware/ww/pkg/runtime/svc/join"
	neighborhood_service "github.com/wetware/ww/pkg/runtime/svc/neighborhood"
	tick_service "github.com/wetware/ww/pkg/runtime/svc/ticker"
	tracker_service "github.com/wetware/ww/pkg/runtime/svc/tracker"
)

const timestep = time.Millisecond * 100

func services() fx.Option {
	return fx.Provide(
		tick_service.New,
		epoch_service.New,
		tracker_service.New,
		neighborhood_service.New,
		boot_service.New,
		beacon_service.New,
		// discover_service.New,
		graph_service.New,
		announcer_service.New,
		join_service.New,
	)
}

// Config for the server runtime.
type Config struct {
	log ww.Logger

	ns         string
	ttl        time.Duration
	kmin, kmax int

	psk   pnet.PSK
	addrs []multiaddr.Multiaddr
	ds    datastore.Batching
	boot  boot.Strategy
}

func (cfg Config) export() fx.Option {
	return fx.Options(
		fx.NopLogger,
		fx.Provide(
			cfg.options,
			p2p.New,
			cluster.New,
			// block.New,
			newAnchor,
			newHost,
		),
		services(),
		fx.Invoke(
			runtime.Start,
			listen,
		),
	)
}

func (cfg Config) options(lx fx.Lifecycle) (mod module, err error) {
	mod.Ctx = ctxutil.WithLifecycle(context.Background(), lx) // libp2p lifecycle
	mod.Log = cfg.log.WithField("ns", cfg.ns)
	mod.Namespace = cfg.ns
	mod.TTL = cfg.ttl
	mod.Boot = cfg.boot
	mod.ListenAddrs = cfg.addrs
	mod.KMin = cfg.kmin
	mod.KMax = cfg.kmax

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
	Log       ww.Logger
	Namespace string        `name:"ns"`
	TTL       time.Duration `name:"ttl"`

	KMin int `name:"kmin"`
	KMax int `name:"kmax"`

	ListenAddrs []multiaddr.Multiaddr
	Boot        boot.Strategy

	HostOpt []config.Option
	DHTOpt  []dual.Option

	Datastore datastore.Batching
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
