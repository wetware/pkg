package client

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/fx"

	// libp2p / ipfs
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-eventbus"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/pnet"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/config"

	// wetware internal
	ctxutil "github.com/wetware/ww/internal/util/ctx"
	hostutil "github.com/wetware/ww/internal/util/host"
	"github.com/wetware/ww/pkg/internal/p2p"

	// wetware public
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/boot"
	"github.com/wetware/ww/pkg/runtime"

	// runtime services
	boot_service "github.com/wetware/ww/pkg/runtime/svc/boot"
	graph_service "github.com/wetware/ww/pkg/runtime/svc/graph"
	join_service "github.com/wetware/ww/pkg/runtime/svc/join"
	neighborhood_service "github.com/wetware/ww/pkg/runtime/svc/neighborhood"
	tick_service "github.com/wetware/ww/pkg/runtime/svc/ticker"
	tracker_service "github.com/wetware/ww/pkg/runtime/svc/tracker"
)

func services() fx.Option {
	return fx.Provide(
		tick_service.New,
		tracker_service.New,
		neighborhood_service.New,
		boot_service.New,
		// discovery_service.New,  // TODO:  initial advertisement
		graph_service.New,
		join_service.New,
	)
}

// Config contains user-supplied parameters used by Dial.
type Config struct {
	log        ww.Logger
	ns         string
	psk        pnet.PSK
	ds         datastore.Batching
	d          boot.Strategy
	kmin, kmax int
}

func (cfg Config) export(ctx context.Context) fx.Option {
	return fx.Options(
		fx.NopLogger,
		fx.Provide(
			cfg.options,
			p2p.New,
			newClient,
		),
		services(),
		fx.Invoke(
			runtime.Start,
			dial,
		),
	)
}

func (cfg Config) options(lx fx.Lifecycle) (mod module, err error) {
	mod.Ctx = ctxutil.WithLifecycle(context.Background(), lx) // libp2p lifecycle
	mod.Log = cfg.log.WithField("ns", cfg.ns)
	mod.Namespace = cfg.ns
	mod.Datastore = cfg.ds
	mod.Boot = cfg.d
	mod.KMin = cfg.kmin
	mod.KMax = cfg.kmax

	// options for host.Host
	mod.HostOpt = []config.Option{
		hostutil.MaybePrivate(cfg.psk),
		libp2p.Ping(false),
		libp2p.NoListenAddrs, // also disables relay
		libp2p.UserAgent("ww-client"),
	}

	// options for DHT
	mod.DHTOpt = append(mod.DHTOpt, dual.DHTOption(
		dht.Datastore(cfg.ds),
		dht.Mode(dht.ModeClient),
	))

	return
}

type module struct {
	fx.Out

	Ctx       context.Context
	Log       ww.Logger
	Namespace string `name:"ns"`
	KMin      int    `name:"kmin"`
	KMax      int    `name:"kmax"`

	Datastore datastore.Batching
	Boot      boot.Strategy

	HostOpt []config.Option
	DHTOpt  []dual.Option
}

func dial(h host.Host, dht routing.Routing, lx fx.Lifecycle) error {
	e, err := h.EventBus().Emitter(new(p2p.EvtNetworkReady), eventbus.Stateful)
	if err != nil {
		return err
	}
	defer e.Close()

	lx.Append(dialhook(h.EventBus(), dht))

	return e.Emit(netready(h))
}

func dialhook(bus event.Bus, dht interface{ Bootstrap(context.Context) error }) fx.Hook {
	return fx.Hook{
		OnStart: func(ctx context.Context) error {
			sub, err := bus.Subscribe(new(neighborhood_service.EvtNeighborhoodChanged))
			if err != nil {
				return err
			}
			defer sub.Close()

			for {
				select {
				case v := <-sub.Out():
					if v.(neighborhood_service.EvtNeighborhoodChanged).To == neighborhood_service.PhaseOrphaned {
						continue
					}

					return errors.Wrap(dht.Bootstrap(ctx), "dht bootstrap") // async call
				case <-ctx.Done():
					return errors.Wrap(ctx.Err(), "join cluster")
				}
			}
		},
	}
}

func netready(h host.Host) p2p.EvtNetworkReady {
	return p2p.EvtNetworkReady{Network: h.Network()}
}
