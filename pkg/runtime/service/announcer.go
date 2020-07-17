package service

import (
	"context"
	"math/rand"
	"time"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/lthibault/jitterbug"
	"github.com/lthibault/wetware/pkg/internal/routing"
	"github.com/lthibault/wetware/pkg/runtime"
	randutil "github.com/lthibault/wetware/pkg/util/rand"
)

// Publisher can publish messages to a pubsub topic.
type Publisher interface {
	Publish(context.Context, []byte, ...pubsub.PubOpt) error
}

// Announcer publishes cluster-wise heartbeats that announces the local host to peers.
//
// Consumes:
//
// Emits:
func Announcer(h host.Host, p Publisher, ttl time.Duration) ProviderFunc {
	return func() (runtime.Service, error) {
		tstep, err := h.EventBus().Subscribe(new(EvtTimestep))
		if err != nil {
			return nil, err
		}

		ctx, cancel := context.WithCancel(context.Background())
		a := announcer{
			h:        h,
			ttl:      ttl,
			p:        p,
			ctx:      ctx,
			cancel:   cancel,
			tstep:    tstep,
			errs:     make(chan error, 1),
			announce: make(chan struct{}),
		}

		return a, nil
	}
}

type announcer struct {
	h   host.Host
	ttl time.Duration
	p   Publisher

	ctx    context.Context
	cancel context.CancelFunc

	tstep    event.Subscription
	errs     chan error
	announce chan struct{}
}

func (a announcer) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"service": "announcer",
		"ttl":     a.ttl,
	}
}

func (a announcer) Start(ctx context.Context) (err error) {
	if err = waitNetworkReady(ctx, a.h.EventBus()); err == nil {
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

func (a announcer) Errors() <-chan error {
	return a.errs
}

func (a announcer) Announce(ctx context.Context) error {
	hb, err := routing.NewHeartbeat(a.h.ID(), a.ttl)
	if err != nil {
		return err
	}

	b, err := routing.MarshalHeartbeat(hb)
	if err != nil {
		return err
	}

	return a.p.Publish(ctx, b)
}

func (a announcer) subloop() {
	defer close(a.announce)

	// Hosts tend to be started in batches, which causes heartbeat storms.  We
	// add a small ammount of jitter to smooth things out.  The jitter is
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
	s := newScheduler(a.ttl/2, jitterbug.Uniform{
		Min:    a.ttl / 4,
		Source: rand.New(randutil.FromPeer(a.h.ID())),
	})

	for v := range a.tstep.Out() {
		if s.Advance(v.(EvtTimestep).Delta) {
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
	defer close(a.errs)

	for range a.announce {
		if err := a.Announce(a.ctx); err != nil && err != context.Canceled {
			select {
			case a.errs <- err:
			case <-a.ctx.Done():
			}
		}
	}
}
