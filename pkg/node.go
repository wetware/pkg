package ww

import (
	"context"
	"io"

	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/lthibault/log"
	"go.uber.org/fx"

	"github.com/wetware/casm/pkg/cluster/pulse"
)

type Config struct {
	fx.In

	NS     string      `name:"ns" optional:"true"`
	Logger log.Logger  `optional:"true"`
	Store  ds.Batching `optional:"true"`
	Host   host.Host
	DHT    DHT
	PubSub PubSub
}

func (cfg Config) Supply() fx.Option {
	return fx.Supply(struct {
		fx.Out

		NS     string `name:"ns"`
		Logger log.Logger
		Store  ds.Batching
		Host   host.Host
		DHT    DHT
		PubSub PubSub
	}{
		NS:     cfg.namespace(),
		Logger: cfg.logger(),
		Store:  cfg.datastore(),
		Host:   cfg.Host,
		DHT:    cfg.DHT,
		PubSub: cfg.PubSub,
	})
}

// Node in a Wetware cluster.
type Node struct {
	ns  string
	log log.Logger

	host   host.Host
	dht    DHT
	pubsub PubSub

	app *fx.App
}

// New node in the Wetware cluster.  The context is used exclusively during
// initialization and can be safely cancelled after 'New' returns.
func New(ctx context.Context, cfg Config) (n Node, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	n.app = fx.New(fx.NopLogger,
		fx.Populate(&n),
		cfg.Supply(),
		fx.Provide(
			eventloop,
			cluster,
			node))

	err = n.app.Start(ctx)
	return
}

func (n Node) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"ns":    n.ns,
		"id":    n.host.ID(),
		"addrs": n.host.Addrs(),
	}
}

// Shutdown leaves the cluster gracefully by publishing a LEAVE message to
// the cluster before exiting.  The context can be used to time-out from
// this operation.  Any error other than 'context.DeadlineExceeded' and
// 'context.Canceled' should be treated a failure to properly release shared
// resources.
func (n Node) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	return n.app.Stop(ctx)
}

type nodeConfig struct {
	fx.In

	NS     string `name:"ns"`
	Logger log.Logger
	Host   host.Host
	DHT    DHT
	PubSub PubSub
}

func node(cfg nodeConfig) Node {
	return Node{
		ns:     cfg.NS,
		log:    cfg.Logger,
		host:   cfg.Host,
		dht:    cfg.DHT,
		pubsub: cfg.PubSub,
	}
}

func (cfg Config) logger() log.Logger {
	return cfg.Logger.
		WithField("ns", cfg.namespace()).
		WithField("id", cfg.Host.ID()).
		WithField("addr", cfg.Host.Addrs())
}

func (cfg Config) namespace() string {
	if cfg.NS != "" {
		return cfg.NS
	}

	return "ww"
}

func (cfg Config) datastore() ds.Batching {
	if cfg.Store != nil {
		return cfg.Store
	}

	return sync.MutexWrap(ds.NewMapDatastore())
}

func eventloop(bus event.Bus, lx fx.Lifecycle) (sub event.Subscription, e event.Emitter, err error) {
	if sub, err = bus.Subscribe([]interface{}{
		new(pulse.EvtMembershipChanged),
		new(event.EvtLocalAddressesUpdated),
	}); err == nil {
		lx.Append(closer(sub))
	}

	if e, err = bus.Emitter(new(pulse.EvtMembershipChanged)); err == nil {
		lx.Append(closer(e))
	}

	return
}

func closer(c io.Closer) fx.Hook {
	return fx.Hook{
		OnStop: func(context.Context) error {
			return c.Close()
		},
	}
}
