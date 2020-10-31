package announcer

import (
	"context"
	"encoding/binary"
	"math/rand"
	"time"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"go.uber.org/fx"

	"github.com/lthibault/jitterbug"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/runtime"
	"github.com/wetware/ww/pkg/runtime/svc/internal"
	tick_service "github.com/wetware/ww/pkg/runtime/svc/ticker"
	randutil "github.com/wetware/ww/pkg/util/rand"
)

// Config for Announcer service.
type Config struct {
	fx.In

	Log           ww.Logger
	Host          host.Host
	Announcements *pubsub.Topic
	TTL           time.Duration `name:"ttl"`
}

// NewService satisfies runtime.ServiceFactory.
func (cfg Config) NewService() (runtime.Service, error) {
	tstep, err := cfg.Host.EventBus().Subscribe(new(tick_service.EvtTimestep))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	a := announcer{
		log:      cfg.Log,
		h:        cfg.Host,
		ttl:      cfg.TTL,
		p:        cfg.Announcements,
		ctx:      ctx,
		cancel:   cancel,
		tstep:    tstep,
		announce: make(chan struct{}),
	}

	return a, nil
}

// Module for Announcer service.
type Module struct {
	fx.Out

	Factory runtime.ServiceFactory `group:"runtime"`
}

// Publisher can publish messages to a pubsub topic.
type Publisher interface {
	Publish(context.Context, []byte, ...pubsub.PubOpt) error
}

// New Announcer service.  Publishes cluster-wise heartbeats that announces the local
// host to peers.
//
// Consumes:
//
// Emits:
func New(cfg Config) Module { return Module{Factory: cfg} }

type announcer struct {
	log ww.Logger

	h   host.Host
	ttl time.Duration
	p   Publisher

	ctx    context.Context
	cancel context.CancelFunc

	tstep    event.Subscription
	announce chan struct{}
}

func (a announcer) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"service": "announcer",
		"ttl":     a.ttl,
	}
}

func (a announcer) Start(ctx context.Context) (err error) {
	if err = internal.WaitNetworkReady(ctx, a.h.EventBus()); err == nil {
		if err = a.Announce(ctx); err == nil {
			go a.subloop()
			go a.announceloop()
		}
	}

	return
}

func (a announcer) Stop(ctx context.Context) error {
	a.cancel()

	return a.tstep.Close()
}

func (a announcer) Announce(ctx context.Context) error {
	b := make([]byte, binary.MaxVarintLen64)
	return a.p.Publish(ctx, b[:binary.PutUvarint(b, uint64(a.ttl))])
}

func (a announcer) subloop() {
	defer close(a.announce)

	// Hosts tend to be started in batches, which causes heartbeat storms.  We
	// add a small amount of jitter to smooth things out.  The jitter is
	// calculated by sampling from a uniform distribution between .25 * TTL and
	// .5 * TTL.  The TTL corresponds to 2.6 heartbeats, on average.
	//
	// With default TTL settings, a heartbeat is emitted every 2250ms, on
	// average.  This tolerance is optimized for the widest possible variety of
	// execution settings, and should notably perform well on high-latency
	// networks, including 3G.
	//
	// Clusters operating in low-latency settings such as datacenters may wish
	// to reduce the TTL.  Doing so will increase the cluster's responsiveness
	// at the expense of an O(n) increase in bandwidth consumption.
	s := internal.NewScheduler(a.ttl/2, jitterbug.Uniform{
		Min:    a.ttl / 4,
		Source: rand.New(randutil.FromPeer(a.h.ID())),
	})

	for v := range a.tstep.Out() {
		if s.Advance(v.(tick_service.EvtTimestep).Delta) {
			select {
			case a.announce <- struct{}{}:
			default:
				// an announcement is in progress
			}

			s.Reset()
		}
	}
}

func (a announcer) announceloop() {
	for range a.announce {
		if err := a.Announce(a.ctx); err != nil && err != context.Canceled {
			a.log.With(a).WithError(err).Warn("announcement failed")
		}
	}
}
