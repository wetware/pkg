package server

import (
	"context"
	"sync"
	"time"

	"go.uber.org/fx"

	"github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	p2p "github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	cm "github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/event"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/pnet"
	"github.com/libp2p/go-libp2p-core/routing"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-peerstore/pstoreds"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/config"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/multiformats/go-multiaddr"

	log "github.com/lthibault/log/pkg"
	hostutil "github.com/lthibault/wetware/internal/util/host"
	"github.com/lthibault/wetware/pkg/cluster"
	discover "github.com/lthibault/wetware/pkg/discover"
	"github.com/lthibault/wetware/pkg/internal/eventloop"
)

/*
	server.go handles dependency injection for `Host`
*/

func module(h *Host, opt []Option) fx.Option {
	return fx.Options(
		fx.NopLogger,
		fx.Supply(opt),

		/**************************
		 *  declare dependencies  *
		 **************************/
		fx.Provide(
			newCtx,
			userConfig,
			newConnMgr,
			newDatastore,
			newPeerstore,
			newRoutedHost,
			newDiscovery,
			newPubSub,
			newRouter,
			newWwHost,
		),

		/*********************************
		 *  build public-facing structs  *
		 *********************************/
		fx.Populate(h),

		/******************************
		 *  start runtime goroutines  *
		 ******************************/
		fx.Invoke(
			bootstrap,
			startEventLoop,
			maintainConnectivity,
			announce,

			// This MUST come last. Fires event.EvtLocalAddressesUpdated.
			listenAndServe,
		),
	)
}

/*
	Constructors
*/

type wwHostConfig struct {
	fx.In

	Log    log.Logger
	Host   host.Host
	Router *cluster.Router
}

func newWwHost(cfg wwHostConfig) Host {
	// logger fields are not available until host starts listening.
	f := cachedLogFactory(func() log.Logger {
		return cfg.Log.WithFields(log.F{
			"id":    cfg.Host.ID(),
			"addrs": cfg.Host.Addrs(),
		})
	})

	registerProtocols(f, cfg.Host, cfg.Router)

	return Host{
		logFactory: f,
		host:       cfg.Host,
		r:          cfg.Router,
	}
}

type routerConfig struct {
	fx.In

	Ctx       context.Context
	Namespace string `name:"ns"`

	PubSub *pubsub.PubSub
}

func newRouter(lx fx.Lifecycle, cfg routerConfig) (*cluster.Router, error) {
	r, err := cluster.NewRouter(cfg.Ctx, cfg.Namespace, cfg.PubSub)
	if err == nil {
		lx.Append(fx.Hook{
			OnStop: func(context.Context) error {
				return r.Close()
			},
		})
	}

	return r, err
}

type pubsubConfig struct {
	fx.In

	Ctx       context.Context
	Host      host.Host
	Discovery discovery.Discovery
}

func newPubSub(lx fx.Lifecycle, cfg pubsubConfig) (*pubsub.PubSub, error) {
	return pubsub.NewGossipSub(
		cfg.Ctx,
		cfg.Host,
		pubsub.WithDiscovery(cfg.Discovery),
	)
}

func newDiscovery(r routing.Routing) discovery.Discovery {
	return discovery.NewRoutingDiscovery(r)
}

type hostConfig struct {
	fx.In

	Ctx       context.Context
	PSK       pnet.PSK
	Peerstore peerstore.Peerstore
	ConnMgr   cm.ConnManager
	Datastore datastore.Batching
}

func (cfg hostConfig) options() []config.Option {
	return []config.Option{
		p2p.DisableRelay(),
		hostutil.MaybePrivate(cfg.PSK),
		p2p.NoListenAddrs, // defer listening until setup is complete
		p2p.UserAgent("ww-host"),
		p2p.Peerstore(cfg.Peerstore),
		p2p.ConnectionManager(cfg.ConnMgr),
	}
}

type hostOut struct {
	fx.Out

	Host      host.Host
	DHT       routing.Routing
	Signaller addrChangeSignaller
}

func newRoutedHost(lx fx.Lifecycle, cfg hostConfig) (out hostOut, err error) {
	if out.Host, err = p2p.New(cfg.Ctx, cfg.options()...); err != nil {
		return
	}

	out.Signaller = out.Host.(addrChangeSignaller)

	lx.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return out.Host.Close()
		},
	})

	out.DHT = dht.NewDHT(cfg.Ctx, out.Host, cfg.Datastore)
	out.Host = routedhost.Wrap(out.Host, out.DHT)

	return
}

type peerstoreConfig struct {
	fx.In

	Ctx    context.Context
	DStore datastore.Batching
}

func newPeerstore(cfg peerstoreConfig) (peerstore.Peerstore, error) {
	return pstoreds.NewPeerstore(cfg.Ctx, cfg.DStore, pstoreds.DefaultOpts())
}

func newDatastore() datastore.Batching {
	// TODO:  `newBlockstore`, with ARC cache.  N.B.:  do it in another constructor
	//		   because you'll need to pass the raw Datastore in other places.
	//
	//		   When is this strictly necessary?  I'm guessing for BitSwap?
	//
	// See `BaseBlockstoreCtor` (Ctor == "constructor") below:
	// https://github.com/ipfs/go-ipfs/blob/b19d57fb62c8cf275edf58c2a41f65c14ebe6295/core/node/storage.go#L30

	return dsync.MutexWrap(datastore.NewMapDatastore())
}

type connManagerConfig struct {
	fx.In

	K clusterCardinality
}

func newConnMgr(cfg connManagerConfig) cm.ConnManager {
	return connmgr.NewConnManager(cfg.K.Min, cfg.K.Max, time.Second*10)
}

type userConfigOut struct {
	fx.Out

	Log log.Logger

	// Network address and cluster joining
	Discover    discover.Protocol
	ListenAddrs []multiaddr.Multiaddr
	Secret      pnet.PSK

	// Pubsub params
	Namespace string        `name:"ns"`
	TTL       time.Duration `name:"ttl"`

	// Neighborhood params
	K clusterCardinality

	// Misc
	EventHandlers []eventloop.Handler
}

func userConfig(opt []Option) (out userConfigOut, err error) {
	var cfg Config

	for _, f := range withDefault(opt) {
		if err = f(&cfg); err != nil {
			return
		}
	}

	out.Log = cfg.log.WithField("ns", cfg.ns)

	out.Discover = cfg.d
	out.ListenAddrs = cfg.addrs

	out.Namespace = cfg.ns
	out.Secret = cfg.psk
	out.TTL = cfg.ttl

	out.K = cfg.k

	out.EventHandlers = cfg.evtHandlers

	return
}

func newCtx(lx fx.Lifecycle) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	lx.Append(fx.Hook{
		OnStop: func(context.Context) error {
			cancel()
			return nil
		},
	})

	return ctx
}

/*
	Runtime processes (use fx.Invoke)
*/

func bootstrap(lx fx.Lifecycle, beacon discover.Protocol, host host.Host) {
	lx.Append(fx.Hook{
		// We must wait until the libp2p.Host is listening before
		// advertising our listen addresses.
		OnStart: func(context.Context) error {
			return beacon.Start(host)
		},
		OnStop: func(context.Context) error {
			return beacon.Close()
		},
	})
}

type listenAndServeConfig struct {
	fx.In

	Host      host.Host
	Addrs     []multiaddr.Multiaddr
	Signaller addrChangeSignaller
}

func listenAndServe(cfg listenAndServeConfig) error {
	sub, err := cfg.Host.EventBus().Subscribe(new(event.EvtLocalAddressesUpdated))
	if err != nil {
		return err
	}

	if err := cfg.Host.Network().Listen(cfg.Addrs...); err != nil {
		return err
	}

	// ensure the host fires event.EvtLocalAddressUpdated immediately.
	cfg.Signaller.SignalAddressChange()

	<-sub.Out()
	return sub.Close()
}

/*
	Misc.
*/

// WARNING: this interface is unstable and may removed from basichost.BasicHost in the
// 		    future.  Hopefully this will only happen after they properly refactor Host
// 			setup.
type addrChangeSignaller interface {
	SignalAddressChange()
}

// logFactory is an interface for lazily configuring structured loggers.
// It is used to configure a logger before its field values are known.
// See the Host constructor for a canonical example.
type logFactory interface {
	Log() log.Logger
}

type logFactoryFunc func() log.Logger

func (f logFactoryFunc) Log() log.Logger {
	return f()
}

// cachedLogFactory returns a provider that calls `f` the first time it is invoked, caches
// the result, and always returns this result.
func cachedLogFactory(f func() log.Logger) logFactoryFunc {
	var once sync.Once
	var cached log.Logger
	return func() log.Logger {
		once.Do(func() { cached = f() })
		return cached
	}
}

type clusterCardinality struct {
	Min, Max int
}
