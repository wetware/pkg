package server

import (
	"context"
	"encoding/binary"
	"math/rand"
	"time"

	"go.uber.org/fx"

	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/jitterbug"
	log "github.com/lthibault/log/pkg"
	randutil "github.com/lthibault/wetware/pkg/util/rand"

	ww "github.com/lthibault/wetware/pkg"
)

type routingtableConfig struct {
	fx.In

	Ctx context.Context
	Log log.Logger

	Host host.Host

	Namespace string        `name:"ns"`
	TTL       time.Duration `name:"ttl"`

	PubSub *pubsub.PubSub
	Filter filter
}

func routingtable(lx fx.Lifecycle, cfg routingtableConfig) (err error) {
	if err = cfg.PubSub.RegisterTopicValidator(cfg.Namespace,
		newHeartbeatValidator(cfg.Ctx, cfg.Filter)); err != nil {
		return
	}

	var topic *pubsub.Topic
	if topic, err = cfg.PubSub.Join(cfg.Namespace); err != nil {
		return err
	}

	lx.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return topic.Close()
		},
	})

	announce(cfg.Ctx, lx, announcer{
		log:    cfg.Log,
		hostID: cfg.Host.ID(),
		ttl:    cfg.TTL,
		mesh:   topic,
	})

	return nil
}

func announce(ctx context.Context, lx fx.Lifecycle, a announcer) {
	ctx, cancel := context.WithCancel(ctx)
	lx.Append(fx.Hook{
		// We must wait until the libp2p.Host is listening before
		// announcing ourself to peers.
		OnStart: func(start context.Context) (err error) {
			if err = a.Announce(start); err == nil {
				go a.loop(ctx)
			}

			return
		},
		OnStop: func(stop context.Context) error {
			cancel()
			return nil
		},
	})
}

type announcer struct {
	log log.Logger

	hostID peer.ID
	ttl    time.Duration

	mesh interface {
		Publish(context.Context, []byte, ...pubsub.PubOpt) error
	}
}

func (a announcer) Announce(ctx context.Context) error {
	hb, err := ww.NewHeartbeat(a.hostID, a.ttl)
	if err != nil {
		return err
	}

	b, err := ww.MarshalHeartbeat(hb)
	if err != nil {
		return err
	}

	return a.mesh.Publish(ctx, b)
}

func (a announcer) loop(ctx context.Context) {
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
	ticker := jitterbug.New(a.ttl/2, jitterbug.Uniform{
		Min:    a.ttl / 4,
		Source: rand.New(randutil.FromPeer(a.hostID)),
	})
	defer ticker.Stop()

	for range ticker.C {
		switch err := a.Announce(ctx); err {
		case nil:
		case context.Canceled:
			return
		default:
			a.log.WithError(err).Error("failed to announce")
		}
	}
}

func newHeartbeatValidator(ctx context.Context, f filter) pubsub.Validator {
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

func seqno(msg *pubsub.Message) uint64 {
	return binary.BigEndian.Uint64(msg.GetSeqno())
}
