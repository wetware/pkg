package runtime

import (
	"context"
	"math/rand"
	"time"

	"github.com/lthibault/jitterbug"
	log "github.com/lthibault/log/pkg"
	"github.com/pkg/errors"
	"go.uber.org/fx"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	ww "github.com/lthibault/wetware/pkg"
	randutil "github.com/lthibault/wetware/pkg/util/rand"
)

type announceParams struct {
	fx.In

	Log          log.Logger
	Host         host.Host
	TTL          time.Duration `name:"ttl"`
	RoutingTopic *pubsub.Topic
}

func announce(ctx context.Context, ps announceParams, lx fx.Lifecycle) error {
	a := clusterAnnouner{
		log:  ps.Log.WithField("service", "announcer"),
		id:   ps.Host.ID(),
		ttl:  ps.TTL,
		mesh: ps.RoutingTopic,
	}

	ctx, cancel := context.WithCancel(ctx)

	lx.Append(fx.Hook{
		// We must wait until the libp2p.Host is listening before
		// announcing ourself to peers.
		OnStart: func(start context.Context) (err error) {
			// We must wait until the libp2p.Host is listening before
			// advertising our listen addresses.  If you encounter this error,
			// try starting the announcer later.
			if len(ps.Host.Addrs()) == 0 {
				return errors.New("start beacon: host is not listening")
			}

			if err = a.Announce(start); err == nil {
				go a.loop(ctx)
			}

			a.log.Debug("service started")
			return
		},
		OnStop: func(stop context.Context) error {
			cancel()
			a.log.Debug("service stopped")
			return nil
		},
	})

	return nil
}

type clusterAnnouner struct {
	log log.Logger

	id  peer.ID
	ttl time.Duration

	mesh interface {
		Publish(context.Context, []byte, ...pubsub.PubOpt) error
	}
}

func (a clusterAnnouner) Announce(ctx context.Context) error {
	hb, err := ww.NewHeartbeat(a.id, a.ttl)
	if err != nil {
		return err
	}

	b, err := ww.MarshalHeartbeat(hb)
	if err != nil {
		return err
	}

	return a.mesh.Publish(ctx, b)
}

func (a clusterAnnouner) loop(ctx context.Context) {
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
		Source: rand.New(randutil.FromPeer(a.id)),
	})
	defer ticker.Stop()

	for range ticker.C {
		switch err := a.Announce(ctx); err {
		case nil:
		case context.Canceled, pubsub.ErrTopicClosed:
			return
		default:
			a.log.WithError(err).Error("failed to announce")
		}
	}
}
