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
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p-peerstore/pstoreds"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/config"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/multiformats/go-multiaddr"

	log "github.com/lthibault/log/pkg"
	hostutil "github.com/lthibault/wetware/internal/util/host"
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
			startDiscover,
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
	Router *router
}

func newWwHost(cfg wwHostConfig) Host {
	// logger fields are not available until host starts listening.
	log := logLazy(func() log.Logger {
		return cfg.Log.WithFields(log.F{
			"id":    cfg.Host.ID(),
			"addrs": cfg.Host.Addrs(),
		})
	})

	exportRootAnchor(log, cfg.Host, cfg.Router)

	return Host{
		log:  log,
		host: cfg.Host,
		r:    cfg.Router,
	}
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

func newDiscovery(r *dual.DHT) discovery.Discovery {
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
	DHT       *dual.DHT
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

	out.DHT, err = dual.New(cfg.Ctx, out.Host, dht.Datastore(cfg.Datastore))
	if err == nil {
		out.Host = routedhost.Wrap(out.Host, out.DHT)
	}

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

func startDiscover(lx fx.Lifecycle, beacon discover.Protocol, host host.Host) {
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

	DHT *dual.DHT
}

func listenAndServe(lx fx.Lifecycle, cfg listenAndServeConfig) error {
	sub, err := cfg.Host.EventBus().Subscribe(new(event.EvtLocalAddressesUpdated))
	if err != nil {
		return err
	}
	defer sub.Close()

	if err := cfg.Host.Network().Listen(cfg.Addrs...); err != nil {
		return err
	}

	// Ensure the host fires event.EvtLocalAddressUpdated immediately.
	cfg.Signaller.SignalAddressChange()

	// Best-effort attempt at ensuring the DHT is booted when `server.New` returns.
	// This is probably not necessary, but can't hurt.
	cfg.DHT.Bootstrap(nil) // `dht.IpfsDHT.Bootstrap` discards the `ctx` param.

	<-sub.Out()
	return nil
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

type clusterCardinality struct {
	Min, Max int
}

type lazyLogger func() log.Logger

func logLazy(f func() log.Logger) log.Logger {
	var once sync.Once
	var l log.Logger
	return lazyLogger(func() log.Logger {
		once.Do(func() { l = f() })
		return l
	})
}

func (l lazyLogger) Fatal(v ...interface{})                          { l().Fatal(v...) }
func (l lazyLogger) Fatalf(fmt string, v ...interface{})             { l().Fatalf(fmt, v...) }
func (l lazyLogger) Fatalln(v ...interface{})                        { l().Fatalln(v...) }
func (l lazyLogger) Trace(v ...interface{})                          { l().Trace(v...) }
func (l lazyLogger) Tracef(fmt string, v ...interface{})             { l().Tracef(fmt, v...) }
func (l lazyLogger) Traceln(v ...interface{})                        { l().Traceln(v...) }
func (l lazyLogger) Debug(v ...interface{})                          { l().Debug(v...) }
func (l lazyLogger) Debugf(fmt string, v ...interface{})             { l().Debugf(fmt, v...) }
func (l lazyLogger) Debugln(v ...interface{})                        { l().Debugln(v...) }
func (l lazyLogger) Info(v ...interface{})                           { l().Info(v...) }
func (l lazyLogger) Infof(fmt string, v ...interface{})              { l().Infof(fmt, v...) }
func (l lazyLogger) Infoln(v ...interface{})                         { l().Infoln(v...) }
func (l lazyLogger) Warn(v ...interface{})                           { l().Warn(v...) }
func (l lazyLogger) Warnf(fmt string, v ...interface{})              { l().Warnf(fmt, v...) }
func (l lazyLogger) Warnln(v ...interface{})                         { l().Warnln(v...) }
func (l lazyLogger) Error(v ...interface{})                          { l().Error(v...) }
func (l lazyLogger) Errorf(fmt string, v ...interface{})             { l().Errorf(fmt, v...) }
func (l lazyLogger) Errorln(v ...interface{})                        { l().Errorln(v...) }
func (l lazyLogger) WithError(err error) log.Logger                  { return l().WithError(err) }
func (l lazyLogger) WithField(name string, v interface{}) log.Logger { return l().WithField(name, v) }
func (l lazyLogger) WithFields(fs log.F) log.Logger                  { return l().WithFields(fs) }
