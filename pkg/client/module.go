package client

import (
	"context"
	"time"

	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"

	ds "github.com/ipfs/go-datastore"
	p2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pkg/errors"

	hostutil "github.com/lthibault/wetware/internal/util/host"
	ww "github.com/lthibault/wetware/pkg"
	"github.com/lthibault/wetware/pkg/boot"
)

func module(c *Client, s boot.Strategy, opt []Option) fx.Option {
	return fx.Options(
		fx.NopLogger,
		fx.Supply(opt, struct{ boot.Strategy }{s}),
		fx.Provide(
			newCtx,
			newConfig,
			newHost,
			newPubsub,
			newDHT,
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
		log:  p.Cfg.Log(),
		host: p.Host,
	}
}

type hostParams struct {
	fx.In

	Ctx context.Context
	Cfg *Config
}

func newHost(lx fx.Lifecycle, p hostParams) host.Host {
	var h = new(struct{ host.Host })

	lx.Append(fx.Hook{
		OnStart: func(context.Context) (err error) {
			h.Host, err = p2p.New(p.Ctx,
				hostutil.MaybePrivate(p.Cfg.PSK),
				p2p.Ping(false),
				p2p.NoListenAddrs, // also disables relay
				p2p.UserAgent(ww.ClientUAgent))
			return
		},
		OnStop: func(context.Context) error {
			return h.Close()
		},
	})

	return h
}

type dhtParams struct {
	fx.In

	Ctx       context.Context
	Host      host.Host
	Datastore ds.Batching
}

func newDHT(lx fx.Lifecycle, p dhtParams) routing.ContentRouting {
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
	DHT  routing.ContentRouting
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

func join(lx fx.Lifecycle, host host.Host, b boot.Strategy) {
	lx.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ps, err := b.DiscoverPeers(ctx)
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

func connect(ctx context.Context, host host.Host, pinfo peer.AddrInfo) func() error {
	return func() error {
		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		return host.Connect(ctx, pinfo)
	}
}
