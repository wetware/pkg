package ww

import (
	"context"
	"math/rand"
	"time"

	"capnproto.org/go/capnp/v3"
	"github.com/jpillora/backoff"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/jitterbug"
	"github.com/lthibault/log"
	"github.com/multiformats/go-multiaddr"
	"go.uber.org/fx"

	"github.com/wetware/casm/pkg/cluster/pulse"
	"github.com/wetware/casm/pkg/cluster/routing"
)

type clusterConfig struct {
	fx.In

	NS        string `name:"ns"`
	Log       log.Logger
	TTL       time.Duration `optional:"true"`
	Host      host.Host
	PubSub    *pubsub.PubSub
	OnPublish pulse.Hook
	Sub       event.Subscription
	Emitter   event.Emitter
}

type clusterModule struct {
	fx.Out

	Routing *routing.Table
	Topic   *pubsub.Topic
}

func cluster(p clusterConfig, lx fx.Lifecycle) (c clusterModule, err error) {
	c.Topic, err = p.PubSub.Join(p.NS)
	if err == nil {
		c.Routing = routing.New()
		lx.Append(hook(p.validator(c)))
		lx.Append(hook(p.ticker(c)))
		lx.Append(hook(p.relay(c)))
		lx.Append(hook(p.monitor(c)))
		lx.Append(hook(p.heartbeat(c)))
		lx.Append(hook(p.eventLogger()))
	}

	return
}

func (cfg clusterConfig) ttl() time.Duration {
	if cfg.TTL == 0 {
		cfg.TTL = time.Second * 6
	}

	return cfg.TTL
}

// clusterValidator constructs and (de)registers the validator that maintains
// the routing table.
type clusterValidator struct {
	ns     string
	pubsub interface {
		RegisterTopicValidator(topic string, val interface{}, opts ...pubsub.ValidatorOpt) error
		UnregisterTopicValidator(topic string) error
	}
	update pubsub.ValidatorEx
}

func (cfg clusterConfig) validator(c clusterModule) clusterValidator {
	return clusterValidator{
		ns:     cfg.NS,
		pubsub: cfg.PubSub,
		update: pulse.NewValidator(c.Routing, cfg.Emitter),
	}
}

func (cv clusterValidator) Starter() func(context.Context) error {
	return func(context.Context) error {
		return cv.pubsub.RegisterTopicValidator(cv.ns, cv.update)
	}
}

func (cv clusterValidator) Stopper() func(context.Context) error {
	return func(context.Context) error {
		return cv.pubsub.UnregisterTopicValidator(cv.ns)
	}
}

// ticker advances the routing table's internal clock
type ticker struct {
	ticker *time.Ticker
	rt     interface{ Advance(time.Time) }
}

func (p clusterConfig) ticker(c clusterModule) ticker {
	return ticker{
		ticker: time.NewTicker(time.Millisecond * 100),
		rt:     c.Routing,
	}
}

func (t ticker) Starter() func(context.Context) error {
	return func(context.Context) error {
		go func() {
			for now := range t.ticker.C {
				t.rt.Advance(now)
			}
		}()
		return nil
	}
}

func (t ticker) Stopper() func(context.Context) error {
	return func(context.Context) error {
		t.ticker.Stop()
		return nil
	}
}

type relay struct {
	topic  *pubsub.Topic
	cancel pubsub.RelayCancelFunc
}

func (cfg clusterConfig) relay(c clusterModule) *relay {
	return &relay{topic: c.Topic}
}

func (r *relay) Starter() func(context.Context) error {
	return func(context.Context) (err error) {
		r.cancel, err = r.topic.Relay()
		return
	}
}

func (r *relay) Stopper() func(context.Context) error {
	return func(context.Context) error {
		r.cancel()
		return nil
	}
}

type monitor struct {
	ctx    context.Context
	cancel context.CancelFunc

	log log.Logger

	rt *routing.Table

	topic   *pubsub.Topic
	handler *pubsub.TopicEventHandler
}

func (p clusterConfig) monitor(c clusterModule) *monitor {
	ctx, cancel := context.WithCancel(context.Background())
	return &monitor{
		ctx:    ctx,
		cancel: cancel,
		log:    p.Log,
		rt:     c.Routing,
		topic:  c.Topic,
	}
}

func (m *monitor) Starter() func(context.Context) error {
	return func(context.Context) error {
		ev, err := pulse.NewClusterEvent(capnp.SingleSegment(nil))
		if err != nil {
			return err
		}

		if m.handler, err = m.topic.EventHandler(); err == nil {
			go m.handle(m.ctx, ev)
		}

		return err
	}
}

func (m *monitor) Stopper() func(context.Context) error {
	return func(context.Context) error {
		m.cancel()
		m.handler.Cancel()
		return nil
	}
}

func (m *monitor) handle(ctx context.Context, ev pulse.ClusterEvent) {
	for {
		pe, err := m.handler.NextPeerEvent(ctx)
		if err != nil {
			return
		}

		// Don't spam the cluster if we already know about
		// the peer.  Others likely know about it already.
		if contains(m.rt, pe.Peer) {
			continue
		}

		switch pe.Type {
		case pubsub.PeerJoin:
			if err := ev.SetJoin(pe.Peer); err != nil {
				m.log.WithError(err).Fatal("error writing to segment")
			}

		case pubsub.PeerLeave:
			if err := ev.SetLeave(pe.Peer); err != nil {
				m.log.WithError(err).Fatal("error writing to segment")
			}
		}

		b, err := ev.MarshalBinary()
		if err != nil {
			m.log.WithError(err).Fatal("error marshalling capnp message")
		}

		if err = m.topic.Publish(ctx, b, withReady); err != nil {
			return
		}
	}
}

func contains(rt *routing.Table, id peer.ID) bool {
	_, ok := rt.Lookup(id)
	return ok
}

type heartbeat struct {
	ctx    context.Context
	cancel context.CancelFunc

	log log.Logger

	topic publisher
	ttl   time.Duration
	hook  pulse.Hook

	event     pulse.ClusterEvent
	heartbeat pulse.Heartbeat
}

func (cfg clusterConfig) heartbeat(c clusterModule) *heartbeat {
	ctx, cancel := context.WithCancel(context.Background())
	return &heartbeat{
		ctx:    ctx,
		cancel: cancel,
		log:    cfg.Log,
		topic:  c.Topic,
		ttl:    cfg.ttl(),
		hook:   cfg.OnPublish,
	}
}

func (hb *heartbeat) Starter() func(context.Context) error {
	return func(ctx context.Context) (err error) {
		hb.event, err = pulse.NewClusterEvent(capnp.SingleSegment(nil))
		if err != nil {
			return
		}

		hb.heartbeat, err = hb.event.NewHeartbeat()
		if err != nil {
			return
		}

		// publish once to join the cluster
		if err = hb.emit(ctx); err == nil {
			go hb.tick()
		}

		return err
	}
}

func (hb *heartbeat) Stopper() func(context.Context) error {
	return func(context.Context) error {
		hb.cancel()
		return nil
	}
}

func (hb *heartbeat) emit(ctx context.Context) error {
	hb.heartbeat.SetTTL(hb.ttl)
	hb.hook(hb.heartbeat)

	b, err := hb.event.MarshalBinary()
	if err != nil {
		return err
	}

	return hb.topic.Publish(ctx, b, withReady)
}

func (hb *heartbeat) tick() {
	log.Debug("started heartbeat loop")
	defer log.Debug("exited heartbeat loop")

	ticker := jitterbug.New(hb.ttl/2, jitterbug.Uniform{
		Min:    hb.ttl / 3,
		Source: rand.New(rand.NewSource(time.Now().UnixNano())),
	})
	defer ticker.Stop()

	var b = loggableBackoff{backoff.Backoff{
		Factor: 2,
		Min:    hb.ttl / 10,
		Max:    hb.ttl * 10,
		Jitter: true,
	}}

	for range ticker.C {
		if err := hb.emit(hb.ctx); err != nil {
			if err == context.Canceled {
				return
			}

			hb.log.WithError(err).
				With(b).
				Warn("failed to emit heartbeat")

			select {
			case <-time.After(b.Duration()):
				hb.log.With(b).Info("resuming")
				continue

			case <-hb.ctx.Done():
				return
			}
		}

		b.Reset()
	}
}

type eventLogger struct {
	log log.Logger
	sub event.Subscription
}

func (cfg clusterConfig) eventLogger() eventLogger {
	return eventLogger{log: cfg.Log, sub: cfg.Sub}
}

func (e eventLogger) Starter() func(context.Context) error {
	return func(context.Context) error {
		go func() {
			defer e.log.Debug("stopping cluster event logger")

			for v := range e.sub.Out() {
				switch ev := v.(type) {
				case pulse.EvtMembershipChanged:
					e.log.With(ev).
						Info("membership changed")

				case event.EvtLocalAddressesUpdated:
					as := make([]multiaddr.Multiaddr, 0, len(ev.Current))
					for _, a := range ev.Current {
						as = append(as, a.Address)
					}

					e.log.WithField("addrs", as).
						Info("local addresses updated")
				}
			}
		}()

		return nil
	}
}

func (e eventLogger) Stopper() func(context.Context) error {
	return func(context.Context) error {
		e.log.Info("left cluster")
		return nil
	}
}

type loggableBackoff struct{ backoff.Backoff }

func (b loggableBackoff) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"attempt": int(b.Attempt()),
		"dur":     b.ForAttempt(b.Attempt()),
		"max_dur": b.Max,
	}
}

type publisher interface {
	Publish(ctx context.Context, data []byte, opts ...pubsub.PubOpt) error
}

var withReady = pubsub.WithReadiness(pubsub.MinTopicSize(1))

type hookable interface {
	Starter() func(context.Context) error
	Stopper() func(context.Context) error
}

func hook(h hookable) fx.Hook {
	return fx.Hook{
		OnStart: h.Starter(),
		OnStop:  h.Stopper(),
	}
}
