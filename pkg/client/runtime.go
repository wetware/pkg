package client

import (
	"context"

	"go.uber.org/fx"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-eventbus"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/pnet"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/config"

	ctxutil "github.com/lthibault/wetware/internal/util/ctx"
	hostutil "github.com/lthibault/wetware/internal/util/host"
	"github.com/lthibault/wetware/pkg/boot"
	"github.com/lthibault/wetware/pkg/internal/p2p"
	"github.com/lthibault/wetware/pkg/runtime"
	"github.com/lthibault/wetware/pkg/runtime/service"
)

const (
	kmin = 3
	kmax = 64
)

func services(cfg serviceConfig) runtime.ServiceBundle {
	return runtime.Bundle(
		service.ConnTracker(cfg.Host),
		service.Neighborhood(cfg.EventBus, kmin, kmax),
		service.Bootstrap(cfg.EventBus, cfg.Boot),
		// service.Discover(cfg.EventBus, cfg.Namespace, cfg.Discovery),
		service.Graph(cfg.Host),
		service.Joiner(cfg.Host),
	)
}

// Config contains user-supplied parameters used by Dial.
type Config struct {
	ns  string
	psk pnet.PSK
	ds  datastore.Batching
	d   boot.Strategy
}

func (cfg Config) assemble(ctx context.Context, c *Client) {
	c.app = fx.New(
		fx.NopLogger,
		fx.Populate(c),
		fx.Provide(
			cfg.options,
			p2p.New,
			services,
			newClient,
		),
		fx.Invoke(
			runtime.Register,
			start,
		),
	)
}

func (cfg Config) options(lx fx.Lifecycle) (mod module, err error) {
	mod.Ctx = ctxutil.WithLifecycle(context.Background(), lx) // libp2p lifecycle
	mod.Namespace = cfg.ns
	mod.Datastore = cfg.ds
	mod.Boot = cfg.d

	// options for host.Host
	mod.HostOpt = []config.Option{
		hostutil.MaybePrivate(cfg.psk),
		libp2p.Ping(false),
		libp2p.NoListenAddrs, // also disables relay
		libp2p.UserAgent("ww-client"),
	}

	// options for DHT
	mod.DHTOpt = []dht.Option{
		dht.Datastore(cfg.ds),
		dht.Mode(dht.ModeClient),
	}

	return
}

type module struct {
	fx.Out

	Ctx       context.Context
	Namespace string `name:"ns"`

	Datastore datastore.Batching
	Boot      boot.Strategy

	HostOpt []config.Option
	DHTOpt  []dht.Option
}

type serviceConfig struct {
	fx.In

	Namespace string `name:"ns"`
	Host      host.Host
	EventBus  event.Bus
	Discovery discovery.Discovery
	Boot      boot.Strategy
}

func start(h host.Host) error {
	e, err := h.EventBus().Emitter(new(p2p.EvtNetworkReady), eventbus.Stateful)
	if err != nil {
		return err
	}
	defer e.Close()

	return e.Emit(p2p.EvtNetworkReady{Network: h.Network()})
}
