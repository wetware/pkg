package client

import (
	"context"

	"go.uber.org/fx"

	"github.com/ipfs/go-datastore"
	p2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/pnet"
	"github.com/libp2p/go-libp2p-core/routing"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/config"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"

	log "github.com/lthibault/log/pkg"
	hostutil "github.com/lthibault/wetware/internal/util/host"
	ww "github.com/lthibault/wetware/pkg"
	"github.com/lthibault/wetware/pkg/boot"
)

func module(c *Client, s boot.Strategy, opt []Option) fx.Option {
	return fx.Options(
		fx.NopLogger,
		fx.Supply(opt, struct {
			fx.Out
			boot.Strategy
		}{Strategy: s}),
		fx.Provide(
			newCtx,
			userConfig,
			newRoutedHost,
			newPubsub,
			newClient,
		),
		fx.Populate(c),
		runtime,
	)
}

type clientConfig struct {
	fx.In

	Ctx context.Context
	Log log.Logger

	Host   host.Host
	PubSub *pubsub.PubSub
	Topic  *pubsub.Topic
}

func newClient(lx fx.Lifecycle, cfg clientConfig) Client {
	return Client{
		log:  cfg.Log,
		host: cfg.Host,
	}
}

type hostConfig struct {
	fx.In

	Ctx       context.Context
	Datastore datastore.Batching
	Secret    pnet.PSK
}

func (cfg hostConfig) options() []config.Option {
	return []config.Option{
		hostutil.MaybePrivate(cfg.Secret),
		p2p.Ping(false),
		p2p.NoListenAddrs, // also disables relay
		p2p.UserAgent(ww.ClientUAgent),
	}
}

type hostOut struct {
	fx.Out

	Host host.Host
	DHT  routing.Routing
}

func newRoutedHost(lx fx.Lifecycle, cfg hostConfig) (out hostOut, err error) {
	if out.Host, err = p2p.New(cfg.Ctx, cfg.options()...); err != nil {
		return
	}

	lx.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return out.Host.Close()
		},
	})

	out.DHT = dht.NewDHTClient(cfg.Ctx, out.Host, cfg.Datastore)
	out.Host = routedhost.Wrap(out.Host, out.DHT)
	return
}

type pubsubConfig struct {
	fx.In

	Ctx       context.Context
	Host      host.Host
	DHT       routing.Routing
	Namespace string `name:"ns"`
}

type pubsubOut struct {
	fx.Out

	PubSub *pubsub.PubSub
	Topic  *pubsub.Topic
}

func newPubsub(lx fx.Lifecycle, cfg pubsubConfig) (out pubsubOut, err error) {
	if out.PubSub, err = pubsub.NewGossipSub(
		cfg.Ctx,
		cfg.Host,
		pubsub.WithDiscovery(discovery.NewRoutingDiscovery(cfg.DHT)),
	); err != nil {
		return
	}

	if err = out.PubSub.RegisterTopicValidator(
		cfg.Namespace,
		newHeartbeatValidator(cfg.Ctx),
	); err != nil {
		return
	}

	out.Topic, err = out.PubSub.Join(cfg.Namespace)
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

type userConfigOut struct {
	fx.Out

	Log       log.Logger
	Namespace string `name:"ns"`
	Secret    pnet.PSK

	Datastore datastore.Batching
}

func userConfig(opt []Option) (out userConfigOut, err error) {
	cfg := new(Config)
	for _, f := range withDefault(opt) {
		if err = f(cfg); err != nil {
			return
		}
	}

	out.Log = cfg.Log()
	out.Namespace = cfg.ns
	out.Secret = cfg.psk
	out.Datastore = cfg.ds

	return
}

func newHeartbeatValidator(ctx context.Context) pubsub.Validator {
	f := newBasicFilter()

	// Return a function that satisfies pubsub.Validator, using the above background
	// task and filter array.
	return func(_ context.Context, pid peer.ID, msg *pubsub.Message) bool {
		hb, err := ww.UnmarshalHeartbeat(msg.GetData())
		if err != nil {
			return false // drop invalid message
		}

		if id := msg.GetFrom(); !f.Upsert(id, seqno(msg), hb.TTL()) {
			return false // drop stale message
		}

		msg.ValidatorData = hb
		return true
	}
}
