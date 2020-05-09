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
	discover "github.com/lthibault/wetware/pkg/discover"
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
			newFilter,
			newConnMgr,
			newDatastore,
			newPeerstore,
			newRoutedHost,
			newDiscovery,
			newPubSub,
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
			eventloop,
			connpolicy,
			announcer,

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
	Filter filter
}

func newWwHost(cfg wwHostConfig) Host {
	var once sync.Once
	l := cfg.Log
	getLog := loggerFunc(func() log.Logger {
		once.Do(func() {
			l = l.WithFields(log.F{
				"id":    cfg.Host.ID(),
				"addrs": cfg.Host.Addrs(),
			})
		})

		return l
	})

	// registerAnchor(getLog, cfg.Host, cfg.Filter)

	return Host{
		logger: getLog,
		host:   cfg.Host,
		rt:     cfg.Filter,
	}
}

type pubsubConfig struct {
	fx.In

	Ctx       context.Context
	Host      host.Host
	Discovery discovery.Discovery

	Namespace string        `name:"ns"`
	TTL       time.Duration `name:"ttl"`
	Filter    filter
}

type pubsubOut struct {
	fx.Out

	PubSub *pubsub.PubSub
	Topic  *pubsub.Topic
}

func newPubSub(lx fx.Lifecycle, cfg pubsubConfig) (out pubsubOut, err error) {
	if out.PubSub, err = pubsub.NewGossipSub(
		cfg.Ctx,
		cfg.Host,
		pubsub.WithDiscovery(cfg.Discovery),
	); err != nil {
		return
	}

	if err = out.PubSub.RegisterTopicValidator(cfg.Namespace,
		newHeartbeatValidator(cfg.Ctx, cfg.Filter)); err != nil {
		return
	}

	if out.Topic, err = out.PubSub.Join(cfg.Namespace); err != nil {
		return
	}

	lx.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return out.Topic.Close()
		},
	})

	return
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

func newFilter() filter {
	return newBasicFilter()
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

	LowWater  int `name:"kmin"`
	HighWater int `name:"kmax"`
}

func newConnMgr(cfg connManagerConfig) cm.ConnManager {
	return connmgr.NewConnManager(cfg.LowWater, cfg.HighWater, time.Second*10)
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
	KMin int `name:"kmin"` // min peer connections to maintain
	KMax int `name:"kmax"` // max peer connections to maintain

	// Misc
	EventHandlers []evtHandler
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

	out.KMin = cfg.kmin
	out.KMax = cfg.kmax

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

func listenAndServe(cfg listenAndServeConfig) (err error) {
	if err = cfg.Host.Network().Listen(cfg.Addrs...); err == nil {
		// ensure the host fires event.EvtLocalAddressUpdated immediately.
		cfg.Signaller.SignalAddressChange()
	}

	return
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

type logger interface {
	Log() log.Logger
}

type loggerFunc func() log.Logger

func (f loggerFunc) Log() log.Logger {
	return f()
}
