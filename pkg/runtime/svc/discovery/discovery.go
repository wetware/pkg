package discover

import (
	"context"
	"math/rand"
	"time"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/lthibault/jitterbug"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/runtime"
	"github.com/wetware/ww/pkg/runtime/svc/boot"
	"github.com/wetware/ww/pkg/runtime/svc/graph"
	"github.com/wetware/ww/pkg/runtime/svc/internal"
	"github.com/wetware/ww/pkg/runtime/svc/ticker"
	randutil "github.com/wetware/ww/pkg/util/rand"
	"go.uber.org/fx"
)

// TODO(config): parametrize (?)
const adTTL = time.Hour * 2

// Config for Boot service.
type Config struct {
	fx.In

	Log       ww.Logger
	Host      host.Host
	Namespace string `name:"ns"`
	Discovery discovery.Discovery
}

// NewService satisfies runtime.ServiceFactory
func (cfg Config) NewService() (_ runtime.Service, err error) {
	ctx, cancel := context.WithCancel(context.Background())

	d := discoverer{
		log:    cfg.Log,
		h:      cfg.Host,
		ns:     cfg.Namespace,
		d:      cfg.Discovery,
		ctx:    ctx,
		cancel: cancel,
		advert: make(chan struct{}),
		disc:   make(chan struct{}),
	}

	if d.sub, err = cfg.Host.EventBus().Subscribe([]interface{}{
		new(ticker.EvtTimestep),
		new(graph.EvtGraftRequested),
	}); err != nil {
		return
	}

	if d.e, err = cfg.Host.EventBus().Emitter(new(boot.EvtPeerDiscovered)); err != nil {
		return
	}

	return d, nil
}

// Produces boot.EvtPeerDiscovered.
func (cfg Config) Produces() []interface{} {
	return []interface{}{
		boot.EvtPeerDiscovered{},
	}
}

// Consumes ticker.EvtTimestep & graph.EvtGraftRequested.
func (cfg Config) Consumes() []interface{} {
	return []interface{}{
		ticker.EvtTimestep{},
		graph.EvtGraftRequested{},
	}
}

// Module for Boot service.
type Module struct {
	fx.Out
	Factory runtime.ServiceFactory `group:"runtime"`
}

// New Discovery service.  Queries the graph for peers.
//
// Consumes:
//  - EvtTimestep
//  - EvtGraftRequested
//
// Emits:
//  - EvtPeerDiscovered
func New(cfg Config) Module { return Module{Factory: cfg} }

type discoverer struct {
	log ww.Logger

	ns string
	h  host.Host
	d  discovery.Discovery

	ctx    context.Context
	cancel context.CancelFunc

	advert, disc chan struct{}
	sub          event.Subscription
	e            event.Emitter
}

func (d discoverer) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"service": "discover",
		"ns":      d.ns,
	}
}

func (d discoverer) Start(ctx context.Context) (err error) {
	if err = internal.WaitNetworkReady(ctx, d.h.EventBus()); err == nil {
		internal.StartBackground(
			d.adloop,
			d.graftloop,
			d.subloop,
		)
	}

	// TODO(bugfix):  advertise namespace; We currently have to wait 90 minutes for the
	//				  initial advertisement to occur.

	return
}

func (d discoverer) Stop(ctx context.Context) error {
	d.cancel()
	return d.sub.Close()
}

func (d discoverer) subloop() {
	defer close(d.advert)
	defer close(d.disc)

	sched := internal.NewScheduler(adTTL, jitterbug.Uniform{
		Min:    time.Minute * 90,
		Source: rand.New(randutil.FromPeer(d.h.ID())),
	})

	for v := range d.sub.Out() {
		switch ev := v.(type) {
		case ticker.EvtTimestep:
			if !sched.Advance(ev.Delta) {
				continue
			}

			sched.Reset()

			select {
			case d.advert <- struct{}{}:
			default:
			}
		case graph.EvtGraftRequested:
			select {
			case d.disc <- struct{}{}:
			default:
			}
		}
	}
}

func (d discoverer) adloop() {
	for range d.advert {
		ctx, cancel := context.WithTimeout(d.ctx, time.Minute*2)
		defer cancel()

		if _, err := d.d.Advertise(ctx, d.ns, discovery.TTL(adTTL)); err != nil {
			d.log.With(d).WithError(err).Warn("failed to advertise")
		}
	}
}

func (d discoverer) graftloop() {
	for range d.disc {
		ctx, cancel := context.WithTimeout(d.ctx, time.Second*30)
		defer cancel()

		// TODO(performance):  investigate ideal limit & consider making it dynamic.
		ch, err := d.d.FindPeers(ctx, d.ns, discovery.Limit(3))
		if err != nil {
			d.log.With(d).WithError(err).Debug("error finding peers")
		}

		for info := range ch {
			if d.h.ID() == info.ID {
				continue
			}

			if err = d.e.Emit(boot.EvtPeerDiscovered(info)); err != nil {
				d.log.With(d).WithError(err).Error("failed to emit EvtPeerDiscovered")
			}
		}
	}
}
