package service

import (
	"context"
	"math/rand"
	"time"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/lthibault/jitterbug"
	"github.com/lthibault/wetware/pkg/runtime"
	randutil "github.com/lthibault/wetware/pkg/util/rand"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

// TODO(config): parametrize (?)
const adTTL = time.Hour * 2

// Discover service queries the graph for peers.
//
// Consumes:
//  - EvtTimestep
//  - EvtGraftRequested
//
// Emits:
//  - EvtPeerDiscovered
func Discover(h host.Host, ns string, d discovery.Discovery) ProviderFunc {
	return func() (_ runtime.Service, err error) {
		ctx, cancel := context.WithCancel(context.Background())

		d := discoverer{
			h:      h,
			ns:     ns,
			d:      d,
			ctx:    ctx,
			cancel: cancel,
			errs:   make(chan error, 1),
			advert: make(chan struct{}),
			disc:   make(chan struct{}),
		}

		if d.tstep, err = h.EventBus().Subscribe(new(EvtTimestep)); err != nil {
			return
		}

		if d.graft, err = h.EventBus().Subscribe(new(EvtGraftRequested)); err != nil {
			return
		}

		if d.e, err = h.EventBus().Emitter(new(EvtPeerDiscovered)); err != nil {
			return
		}

		return d, nil
	}
}

type discoverer struct {
	ns string
	h  host.Host
	d  discovery.Discovery

	ctx    context.Context
	cancel context.CancelFunc

	errs         chan error
	advert, disc chan struct{}
	tstep, graft event.Subscription
	e            event.Emitter
}

func (d discoverer) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"service": "discover",
		"ns":      d.ns,
	}
}

func (d discoverer) Errors() <-chan error {
	return d.errs
}

func (d discoverer) Start(ctx context.Context) (err error) {
	if err = waitNetworkReady(ctx, d.h.EventBus()); err == nil {
		startBackground(
			d.tsloop,
			d.adloop,
			d.graftloop,
			d.discloop,
		)
	}

	// TODO(bugfix):  advertise namespace; We currently have to wait 90 minutes for the
	//				  initial advertisement to occur.

	return
}

func (d discoverer) Stop(ctx context.Context) error {
	defer close(d.errs)

	d.cancel()

	return multierr.Combine(
		d.tstep.Close(),
		d.graft.Close(),
	)
}

func (d discoverer) tsloop() {
	defer close(d.advert)

	sched := newScheduler(adTTL, jitterbug.Uniform{
		Min:    time.Minute * 90,
		Source: rand.New(randutil.FromPeer(d.h.ID())),
	})

	for v := range d.tstep.Out() {
		if sched.Advance(v.(EvtTimestep).Delta) {
			select {
			case d.advert <- struct{}{}:
				sched.Reset()
			default:
				// advertisement in progress
			}

		}
	}
}

func (d discoverer) graftloop() {
	defer close(d.disc)

	for range d.graft.Out() {
		select {
		case d.disc <- struct{}{}:
		default:
			// discovery in progress
		}
	}
}

func (d discoverer) adloop() {
	for range d.advert {
		d.raise(func() error {
			ctx, cancel := context.WithTimeout(d.ctx, time.Minute*2)
			defer cancel()

			return d.advertise(ctx)
		}())
	}
}

func (d discoverer) discloop() {

	for range d.disc {
		d.raise(func() error {
			ctx, cancel := context.WithTimeout(d.ctx, time.Second*30)
			defer cancel()

			// TODO(performance):  investigate ideal limit & consider making it dynamic.
			ch, err := d.d.FindPeers(ctx, d.ns, discovery.Limit(3))
			if err != nil {
				return errors.Wrap(err, "find peers")
			}

			for info := range ch {
				if d.h.ID() == info.ID {
					continue
				}

				if err = d.e.Emit(EvtPeerDiscovered(info)); err != nil {
					return errors.Wrap(err, "emit")
				}
			}

			return nil
		}())
	}
}

func (d discoverer) advertise(ctx context.Context) (err error) {
	_, err = d.d.Advertise(ctx, d.ns, discovery.TTL(adTTL))
	return
}

func (d discoverer) raise(err error) {
	if err == nil {
		return
	}

	select {
	case d.errs <- err:
	case <-d.ctx.Done():
	}
}
