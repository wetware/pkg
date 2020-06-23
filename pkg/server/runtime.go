package server

import (
	"context"
	"time"

	"github.com/ipfs/go-datastore"
	log "github.com/lthibault/log/pkg"
	"go.uber.org/fx"

	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/pnet"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-peerstore/pstoreds"
	"github.com/libp2p/go-libp2p/config"
	"github.com/multiformats/go-multiaddr"

	ctxutil "github.com/lthibault/wetware/internal/util/ctx"
	hostutil "github.com/lthibault/wetware/internal/util/host"
	discover "github.com/lthibault/wetware/pkg/discover"
	"github.com/lthibault/wetware/pkg/internal/block"
	"github.com/lthibault/wetware/pkg/internal/p2p"
	"github.com/lthibault/wetware/pkg/internal/runtime"
	"github.com/lthibault/wetware/pkg/routing"
)

// Config for the server runtime.
type Config struct {
	log log.Logger

	ns  string
	ttl time.Duration
	gp  runtime.GraphParams

	psk   pnet.PSK
	addrs []multiaddr.Multiaddr
	ds    datastore.Batching
	d     discover.Protocol
}

func (cfg Config) assemble(h *Host) {
	h.app = fx.New(
		fx.NopLogger,
		fx.Populate(h),
		fx.Provide(
			cfg.options,
			p2p.New,
			routing.New,
			block.New,
			newHost,
		),
		runtime.HostEnv(),
	)
}

func (cfg Config) options(lx fx.Lifecycle) (mod module, err error) {
	mod.Ctx = ctxutil.WithLifecycle(context.Background(), lx) // libp2p lifecycle
	mod.Log = cfg.log.WithField("ns", cfg.ns)
	mod.Namespace = cfg.ns
	mod.TTL = cfg.ttl
	mod.Discover = cfg.d
	mod.ListenAddrs = cfg.addrs
	mod.Graph = cfg.gp

	var ps peerstore.Peerstore
	if ps, err = pstoreds.NewPeerstore(mod.Ctx, cfg.ds, pstoreds.DefaultOpts()); err != nil {
		return
	}

	cm := connmgr.NewConnManager(cfg.gp.MinNeighbors, cfg.gp.MaxNeighbors, time.Second*10)

	mod.HostOpt = []config.Option{
		libp2p.DisableRelay(),
		hostutil.MaybePrivate(cfg.psk),
		libp2p.NoListenAddrs, // defer listening until setup is complete
		libp2p.UserAgent("ww-host"),
		libp2p.Peerstore(ps),
		libp2p.ConnectionManager(cm),
	}

	mod.DHTOpt = []dht.Option{
		dht.Datastore(cfg.ds),
		dht.Mode(dht.ModeServer),
	}

	return
}

type module struct {
	fx.Out

	Ctx       context.Context
	Log       log.Logger
	Namespace string        `name:"ns"`
	TTL       time.Duration `name:"ttl"`

	Graph runtime.GraphParams

	ListenAddrs []multiaddr.Multiaddr
	Discover    discover.Protocol

	HostOpt []config.Option
	DHTOpt  []dht.Option

	Datastore datastore.Batching
}
