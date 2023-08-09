//go:generate mockgen -source=cluster.go -destination=../internal/mock/cluster/cluster.go -package=mock_cluster

// Package cluster exports an asynchronously updated model of the swarm.
package cluster

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"capnproto.org/go/capnp/v3"
	"github.com/jpillora/backoff"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/lthibault/jitterbug/v2"
	"github.com/lthibault/log"

	"github.com/wetware/ww/cluster/pulse"
	"github.com/wetware/ww/cluster/routing"
	"github.com/wetware/ww/pkg/view"
)

var ErrClosing = errors.New("closing")

type Topic interface {
	String() string
	Publish(context.Context, []byte, ...pubsub.PubOpt) error
	Relay() (pubsub.RelayCancelFunc, error)
}

// RoutingTable tracks the liveness of cluster peers and provides a
// simple API for querying routing information.
type RoutingTable interface {
	Advance(time.Time)
	Upsert(routing.Record) (created bool)
	Snapshot() routing.Snapshot
}

// Router is a peer participating in the cluster membership protocol.
// It maintains a global view of the cluster with PA/EL guarantees,
// and periodically announces its presence to others.
type Router struct {
	Topic Topic

	Log          log.Logger
	TTL          time.Duration
	Meta         pulse.Preparer
	Clock        Clock
	RoutingTable RoutingTable

	mu             sync.Mutex
	init, relaying atomic.Bool
	id             uint64 // instance ID
	announce       chan []pubsub.PubOpt
	wc             *capnp.WeakClient
}

func (r *Router) Stop() {
	if r.relaying.Swap(true) {
		r.Clock.Stop()
	}
}

func (r *Router) String() string {
	return r.Topic.String()
}

func (r *Router) ID() (id routing.ID) {
	r.setup()
	return routing.ID(r.id)
}

func (r *Router) Loggable() map[string]any {
	return map[string]any{
		"server": r.ID(),
		"ttl":    r.TTL,
		"ns":     r.String(),
	}
}

func (r *Router) View() view.View {
	r.setup()

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.wc != nil {
		if client, ok := r.wc.AddRef(); ok {
			return view.View(client)
		}
	}

	client := view.Server{RoutingTable: r.RoutingTable}.Client()
	r.wc = client.WeakRef()
	return view.View(client)
}

func (r *Router) Bootstrap(ctx context.Context, opt ...pubsub.PubOpt) (err error) {
	if err = r.relay(); err == nil {
		if err = ErrClosing; r.Clock.Context().Err() == nil {
			select {
			case r.announce <- opt:
				err = nil

			case <-r.Clock.Context().Done():
				err = ErrClosing

			case <-ctx.Done():
				err = ctx.Err()
			}
		}
	}

	return
}

// Start relaying messages.  Note that this will not populate
// the routing table unless pulse.Validator was previously set.
func (r *Router) relay() (err error) {
	if r.setup(); !r.relaying.Load() {
		r.mu.Lock()
		defer r.mu.Unlock()

		if !r.relaying.Swap(true) {
			var cancel pubsub.RelayCancelFunc
			if cancel, err = r.Topic.Relay(); err == nil {
				r.Log = r.Log.With(r)
				go r.advance(cancel)
				go r.heartbeat()
			}
		}
	}

	return
}

func (r *Router) setup() {
	if r.init.Load() {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.init.Swap(true) {
		if r.Log == nil {
			r.Log = log.New()
		}

		if r.RoutingTable == nil {
			r.RoutingTable = routing.New(time.Now())
		}

		if r.Meta == nil {
			r.Meta = nopPreparer{}
		}

		if r.TTL <= 0 {
			r.TTL = pulse.DefaultTTL
		}

		if r.Clock == nil {
			r.Clock = NewClock(time.Second)
		}

		r.id = rand.Uint64()
		r.announce = make(chan []pubsub.PubOpt)
	}
}

func (r *Router) advance(cancel pubsub.RelayCancelFunc) {
	defer close(r.announce)
	defer cancel()

	var (
		// jitter between announcements
		jitter = jitterbug.Uniform{
			Min:    r.TTL/2 - 1,
			Source: rand.New(rand.NewSource(time.Now().UnixNano())),
		}

		// next announcement
		next = time.Now()
	)

	ticks := r.Clock.Tick()
	defer r.Clock.Stop()

	for {
		select {
		case t := <-ticks:
			if r.RoutingTable.Advance(t); t.After(next) {
				select {
				case r.announce <- nil:
				default:
				}

				next = t.Add(jitter.Jitter(r.TTL))
			}

		case <-r.Clock.Context().Done():
			return
		}
	}
}

func (r *Router) heartbeat() {
	backoff := &loggableBackoff{backoff.Backoff{
		Factor: 2,
		Min:    r.TTL / 2,
		Max:    time.Minute * 15,
		Jitter: true,
	}}

	hb := pulse.NewHeartbeat()
	hb.SetTTL(r.TTL)
	hb.SetServer(r.ID())

	for a := range r.announce {
		err := r.emit(r.Clock.Context(), hb, a)
		if err == nil {
			backoff.Reset()
			continue
		}

		// shutting down?
		if err == context.Canceled {
			return
		}

		r.Log.
			With(backoff).
			WithError(err).
			Warn("failed to announce")

		// back off...
		select {
		case <-time.After(backoff.Duration()):
			r.Log.Debug("resuming")
		case <-r.Clock.Context().Done():
			return
		}
	}
}

func (r *Router) emit(ctx context.Context, hb pulse.Heartbeat, opt []pubsub.PubOpt) error {
	if err := r.Meta.Prepare(hb); err != nil {
		return err
	}

	msg, err := hb.Message().MarshalPacked()
	if err != nil {
		return err
	}

	return r.Topic.Publish(ctx, msg, opt...)
}

type loggableBackoff struct{ backoff.Backoff }

func (b *loggableBackoff) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"attempt": int(b.Attempt()),
		"dur":     b.ForAttempt(b.Attempt()),
		"max_dur": b.Max,
	}
}

type nopPreparer struct{}

func (nopPreparer) Prepare(pulse.Heartbeat) error { return nil }
