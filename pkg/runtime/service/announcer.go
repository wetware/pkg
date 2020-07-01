package service

import (
	"context"
	"math/rand"
	"time"

	"github.com/lthibault/jitterbug"

	"github.com/libp2p/go-libp2p-core/host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/lthibault/wetware/pkg/internal/routing"
	"github.com/lthibault/wetware/pkg/runtime"
	randutil "github.com/lthibault/wetware/pkg/util/rand"
)

// Announcer publishes cluster-wise heartbeats that announces the local host to peers.
//
// Consumes:
//
// Emits:
func Announcer(h host.Host, t *pubsub.Topic, ttl time.Duration) ProviderFunc {
	return func() (runtime.Service, error) {
		ctx, cancel := context.WithCancel(context.Background())
		a := &announcer{
			h:      h,
			ttl:    ttl,
			t:      t,
			ctx:    ctx,
			cancel: cancel,
			errs:   make(chan error, 1),
		}

		return a, nil
	}
}

type announcer struct {
	h   host.Host
	ttl time.Duration
	t   *pubsub.Topic

	ctx    context.Context
	cancel context.CancelFunc
	ticker *jitterbug.Ticker
	errs   chan error
}

func (a announcer) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"service": "announcer",
		"ttl":     a.ttl,
	}
}

func (a *announcer) Start(ctx context.Context) (err error) {
	if err = waitNetworkReady(ctx, a.h.EventBus()); err != nil {
		return
	}

	if err = a.Announce(ctx); err == nil {
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
		a.ticker = jitterbug.New(a.ttl/2, jitterbug.Uniform{
			Min:    a.ttl / 4,
			Source: rand.New(randutil.FromPeer(a.h.ID())),
		})

		go a.loop()
	}

	return
}

func (a announcer) Stop(ctx context.Context) error {
	defer close(a.errs)
	defer a.cancel()
	defer a.ticker.Stop()

	return nil
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

	return a.t.Publish(ctx, b)
}

func (a *announcer) loop() {
	for range a.ticker.C {
		if err := a.Announce(a.ctx); err != nil && err != context.Canceled {
			a.errs <- err
		}
	}
}
