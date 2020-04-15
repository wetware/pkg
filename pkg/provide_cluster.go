package ww

import (
	"context"
	"encoding/binary"
	"math/rand"
	"time"

	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/lthibault/jitterbug"
	service "github.com/lthibault/service/pkg"
	randutil "github.com/lthibault/wetware/pkg/util/rand"
)

func provideCluster(r *Runtime) service.Service {
	return service.Array{
		provideStreamHandlers(r), // registers stream handlers for ww
		provideHeartbeat(r),
	}
}

func provideStreamHandlers(r *Runtime) service.Service {
	return service.Hook{
		OnStart: func() error {
			r.node.PeerHost.SetStreamHandler("test", func(s network.Stream) {
				r.log.
					WithField("proto", "test").
					WithField("stat", s.Stat()).
					Info("stream handled")

				s.Reset()
			})

			return nil
		},
	}
}

func provideHeartbeat(r *Runtime) service.Service {
	return service.Array{
		filtloop(r),
		subloop(r), // process incoming heartbeat messages
		publoop(r), // emit heartbeat messages
	}
}

func filtloop(r *Runtime) service.Service {
	for i := range r.fs {
		r.fs[i] = newFilter(r.ttl)
	}

	cq := make(chan struct{})
	return service.Hook{
		OnStart: func() error {
			go advance(r, cq)
			return nil
		},
		OnStop: func() error {
			close(cq)
			return nil
		},
	}
}

func subloop(r *Runtime) service.Service {

	var sub iface.PubSubSubscription
	var cancel context.CancelFunc

	return service.Hook{
		OnStart: func() (err error) {
			var ctx context.Context
			ctx, cancel = context.WithCancel(r.ctx)

			if sub, err = heartbeatSubscription(ctx, r); err != nil {
				return
			}

			return handleHeartbeat(ctx, r, sub)
		},
		OnStop: func() error {
			cancel()
			return sub.Close()
		},
	}
}

func publoop(r *Runtime) service.Service {
	var ticker *jitterbug.Ticker
	var cancel context.CancelFunc
	return service.Hook{
		OnStart: func() error {
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
			ticker = jitterbug.New(r.ttl/2, jitterbug.Uniform{
				Min:    r.ttl / 4,
				Source: rand.New(randutil.FromPeer(r.node.PeerHost.ID())),
			})

			var ctx context.Context
			ctx, cancel = context.WithCancel(r.ctx)
			go heartbeatPubLoop(ctx, r, ticker.C)

			return nil
		},
		OnStop: func() error {
			cancel()
			ticker.Stop()
			return nil
		},
	}
}

/*
	helper functions for heartbeat
*/

func handleHeartbeat(ctx context.Context, r *Runtime, sub iface.PubSubSubscription) error {
	emitter, err := heartbeatEmitter(r)
	if err != nil {
		return err
	}

	go func() {
		defer emitter.Close()

		for {
			switch msg, err := sub.Next(ctx); err {
			case nil:
				if stale(r, msg) {
					continue
				}

				hb, err := unmarshalHeartbeat(msg.Data())
				if err != nil {
					r.log.WithError(err).Debug("malformed heartbeat")
					continue
				}

				event, err := hb.ToEvent()
				if err != nil {
					r.log.WithError(err).Debug("heartbeat conversion to event failed")
					continue
				}

				if err = emitter.Emit(event); err != nil {
					panic(err) // Emit doesn't error unless closed
				}
			case context.Canceled:
				return
			default:
				r.log.WithError(err).Debug("error receiving heartbeat")
			}

		}
	}()

	return nil
}

func heartbeatPubLoop(ctx context.Context, r *Runtime, tick <-chan time.Time) {
	for range tick {
		b, err := nextHeartbeat(r)
		if err != nil {
			r.log.WithError(err).Error("failed to create heartbeat message")
			continue
		}

		switch err := r.api.PubSub().Publish(ctx, r.ns, b); err {
		case nil, context.Canceled:
		default:
			// N.B.:  this might also happen if the topic was somehow closed.
			r.log.WithError(err).Fatal("failed to publish (is pubsub enabled?)")
		}
	}
}

func heartbeatSubscription(ctx context.Context, r *Runtime) (iface.PubSubSubscription, error) {
	// TODO:  can we add a custom validator?
	return r.api.PubSub().Subscribe(ctx, r.ns,
		options.PubSub.Discover(true))
}

func heartbeatEmitter(r *Runtime) (event.Emitter, error) {
	bus := r.node.PeerHost.EventBus()
	return bus.Emitter(new(EvtHeartbeat))
}

func nextHeartbeat(r *Runtime) ([]byte, error) {
	hb, err := newHeartbeat(r.node.PeerHost, r.ttl)
	if err != nil {
		return nil, err
	}

	return marshalHeartbeat(hb)
}

/*
	filter helper functions
*/

func advance(r *Runtime, done <-chan struct{}) {
	ticker := time.NewTicker(time.Millisecond * 10)
	defer ticker.Stop()

	for {
		select {
		case t := <-ticker.C:
			for _, f := range r.fs {
				f.Advance(t)
			}
		case <-done:
			return
		}
	}
}

func stale(r *Runtime, msg iface.PubSubMessage) bool {
	f := r.fs[getfidx(msg.From())]
	return !f.Upsert(msg.From(), seqno(msg))
}

func seqno(msg iface.PubSubMessage) uint64 {
	return binary.BigEndian.Uint64(msg.Seq())
}

func getfidx(id peer.ID) int {
	return int(id[len(id)-1])
}
