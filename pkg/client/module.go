package client

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"

	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	p2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

func module(c *Client, d Discover, opt []Option) fx.Option {
	return fx.Options(
		fx.NopLogger,
		fx.Supply(opt, struct{ Discover }{d}),
		fx.Provide(
			newCtx,
			newConfig,
			newHost,
			newDataStore,
			newDHT,
			newPubsub,
			newClient,
		),
		fx.Invoke(join),
		fx.Populate(c),
	)
}

type clientParams struct {
	fx.In

	Ctx  context.Context
	Cfg  *Config
	Host host.Host
	P    *struct{ *pubsub.PubSub }
	T    *struct{ *pubsub.Topic }
}

func newClient(lx fx.Lifecycle, p clientParams) Client {
	for _, hook := range []fx.Hook{
		subloop(p.Ctx, p.Host, p.T),
	} {
		lx.Append(hook)
	}

	return Client{
		host: p.Host,
	}
}

type hostParams struct {
	fx.In

	Ctx      context.Context
	Cfg      *Config
	Discover struct{ Discover }
}

func newHost(lx fx.Lifecycle, p hostParams) host.Host {
	var h = new(struct{ host.Host })

	lx.Append(fx.Hook{
		OnStart: func(context.Context) (err error) {
			h.Host, err = p2p.New(p.Ctx,
				p.Cfg.maybePSK(),
				p2p.Ping(false),
				p2p.NoListenAddrs, // also disables relay
				p2p.UserAgent("ww client"))
			return
		},
		OnStop: func(context.Context) error {
			return h.Close()
		},
	})

	return h
}

func newDataStore() ds.Batching {
	// TODO:  replace this with a more efficient immutable map
	return dsync.MutexWrap(ds.NewMapDatastore())
}

type dhtParams struct {
	fx.In

	Ctx       context.Context
	Host      host.Host
	Datastore ds.Batching
}

func newDHT(lx fx.Lifecycle, p dhtParams) routing.Routing {
	var r = new(struct{ *dht.IpfsDHT })

	lx.Append(fx.Hook{
		OnStart: func(context.Context) error {
			r.IpfsDHT = dht.NewDHTClient(p.Ctx, p.Host, p.Datastore)
			return nil
		},
	})

	return r
}

type pubsubParam struct {
	fx.In

	Ctx  context.Context
	Cfg  *Config
	Host host.Host
	DHT  routing.Routing
}

type pubsubOut struct {
	fx.Out

	P *struct{ *pubsub.PubSub }
	T *struct{ *pubsub.Topic }
}

func newPubsub(lx fx.Lifecycle, p pubsubParam) pubsubOut {
	out := pubsubOut{
		P: new(struct{ *pubsub.PubSub }),
		T: new(struct{ *pubsub.Topic }),
	}

	lx.Append(fx.Hook{
		OnStart: func(context.Context) (err error) {
			out.P.PubSub, err = pubsub.NewGossipSub(p.Ctx, p.Host,
				pubsub.WithDiscovery(discovery.NewRoutingDiscovery(p.DHT)))
			return
		},
	})

	lx.Append(fx.Hook{
		OnStart: func(context.Context) error {

			return out.P.RegisterTopicValidator(p.Cfg.ns,
				newHeartbeatValidator(p.Ctx))
		},
	})

	lx.Append(fx.Hook{
		OnStart: func(context.Context) (err error) {
			out.T.Topic, err = out.P.Join(p.Cfg.ns)
			return
		},
	})

	return out
}

func join(lx fx.Lifecycle, host host.Host, d struct{ Discover }) {
	lx.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ps, err := d.DiscoverPeers(ctx)
			if err != nil {
				return errors.Wrap(err, "discover")
			}

			// TODO:  change this to an at-least-one-succeeds group
			var g errgroup.Group
			for _, pinfo := range ps {
				g.Go(connect(ctx, host, pinfo))
			}

			return errors.Wrap(g.Wait(), "join")
		},
	})
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

func connect(ctx context.Context, host host.Host, pinfo peer.AddrInfo) func() error {
	return func() error {
		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		return host.Connect(ctx, pinfo)
	}
}
