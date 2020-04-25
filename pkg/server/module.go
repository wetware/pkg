package server

import (
	"context"
	"time"

	"go.uber.org/fx"

	"github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	p2p "github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	cm "github.com/libp2p/go-libp2p-core/connmgr"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/routing"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-peerstore/pstoreds"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/config"

	log "github.com/lthibault/log/pkg"
	hostutil "github.com/lthibault/wetware/internal/util/host"
	ww "github.com/lthibault/wetware/pkg"
	"github.com/lthibault/wetware/pkg/boot"
)

func module(h *Host, opt []Option) fx.Option {
	return fx.Options(
		fx.NopLogger,
		fx.Supply(opt),
		fx.Provide(
			newCtx,
			newConfig,
			newConnMgr,
			newDatastore,
			newPeerstore,
			newPeer,
			newDHT,
			newDiscovery,
			newCluster,
			newHost,
		),
		fx.Populate(h),
		fx.Invoke(run),
	)
}

type hostParam struct {
	fx.In

	Log  log.Logger
	Host host.Host
}

func newHost(p hostParam) Host {
	return Host{
		log:  p.Log,
		host: p.Host,
	}
}

type clusterParams struct {
	fx.In

	Ctx       context.Context
	Host      host.Host
	Discovery routingDiscovery

	Namespace string
	TTL       time.Duration
}

type clusterOut struct {
	fx.Out

	PubSub *struct{ *pubsub.PubSub }
	Topic  *struct{ *pubsub.Topic }
}

func newCluster(lx fx.Lifecycle, p clusterParams) (out clusterOut) {
	out.PubSub = new(struct{ *pubsub.PubSub })
	out.Topic = new(struct{ *pubsub.Topic })

	lx.Append(fx.Hook{
		OnStart: func(context.Context) (err error) {
			out.PubSub.PubSub, err = pubsub.NewGossipSub(
				p.Ctx,
				p.Host,
				pubsub.WithDiscovery(p.Discovery),
			)
			return
		},
	})

	lx.Append(fx.Hook{
		OnStart: func(context.Context) (err error) {
			out.Topic.Topic, err = out.PubSub.Join(p.Namespace)
			return
		},
	})

	return
}

func newDiscovery(lx fx.Lifecycle, r routing.ContentRouting) routingDiscovery {
	var rd = new(struct{ *discovery.RoutingDiscovery })

	lx.Append(fx.Hook{
		OnStart: func(context.Context) error {
			rd.RoutingDiscovery = discovery.NewRoutingDiscovery(r)
			return nil
		},
	})

	return rd
}

type dhtParams struct {
	fx.In

	Ctx       context.Context
	Host      host.Host
	Datastore datastore.Batching
}

func newDHT(lx fx.Lifecycle, p dhtParams) routing.ContentRouting {
	var r = new(struct{ *dht.IpfsDHT })

	lx.Append(fx.Hook{
		OnStart: func(context.Context) error {
			r.IpfsDHT = dht.NewDHT(p.Ctx, p.Host, p.Datastore)
			return nil
		},
	})

	return r
}

type peerParams struct {
	fx.In

	Ctx     context.Context
	HostOpt []config.Option
}

func newPeer(lx fx.Lifecycle, p peerParams) host.Host {
	var h = new(struct{ host.Host })

	lx.Append(fx.Hook{
		OnStart: func(context.Context) (err error) {
			h.Host, err = p2p.New(p.Ctx, p.HostOpt...)

			return
		},
		OnStop: func(context.Context) error {
			return h.Close()
		},
	})

	return h
}

type peerstoreParams struct {
	fx.In

	Ctx    context.Context
	DStore datastore.Batching
}

func newPeerstore(p peerstoreParams) (peerstore.Peerstore, error) {
	return pstoreds.NewPeerstore(p.Ctx, p.DStore, pstoreds.DefaultOpts())
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

func newConnMgr() cm.ConnManager {
	return connmgr.NewConnManager(
		ww.LowWater,
		ww.HighWater,
		time.Second*10)
}

type configParams struct {
	fx.In

	Opt     []Option
	PStore  peerstore.Peerstore
	ConnMgr cm.ConnManager
}

type configOutput struct {
	fx.Out

	Log          log.Logger
	Namespace    string
	TTL          time.Duration
	HostOpt      []config.Option
	BootProtocol boot.Protocol
}

func newConfig(p configParams) (out configOutput, err error) {
	var cfg Config

	for _, f := range withDefault(p.Opt) {
		if err = f(&cfg); err != nil {
			return
		}
	}

	out.Log = cfg.log.WithField("ns", cfg.ns)
	out.BootProtocol = cfg.boot
	out.Namespace = cfg.ns
	out.TTL = cfg.ttl

	out.HostOpt = []config.Option{
		p2p.DisableRelay(),
		hostutil.MaybePrivate(cfg.psk),
		p2p.ListenAddrs(cfg.addrs...),
		p2p.UserAgent("ww-host"),
		p2p.Peerstore(p.PStore),
		p2p.ConnectionManager(p.ConnMgr),
	}

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

type routingDiscovery interface {
	discovery.Discovery
	routing.ContentRouting
}
