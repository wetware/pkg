package server

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/lthibault/jitterbug"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"go.uber.org/fx"
	capnp "zombiezen.com/go/capnproto2"

	"github.com/lthibault/wetware/internal/api"
	ww "github.com/lthibault/wetware/pkg"
	randutil "github.com/lthibault/wetware/pkg/util/rand"
)

func announce(ctx context.Context, cfg *Config, p publisher, h hostinfo) fx.Hook {
	a := announcer{
		ns:        cfg.ns,
		ttl:       cfg.ttl,
		host:      h,
		publisher: p,
	}

	ctx, cancel := context.WithCancel(ctx)
	return fx.Hook{
		OnStart: func(start context.Context) (err error) {
			if err = a.Announce(start); err == nil {
				go a.loop(ctx)
			}

			return
		},
		OnStop: func(stop context.Context) error {
			if err := a.Denounce(stop); err != nil {
				cfg.Log().WithError(err).Warn("failed to send GOAWAY")
			}

			cancel()
			return nil
		},
	}
}

type announcer struct {
	ns   string
	ttl  time.Duration
	host hostinfo
	publisher
}

func (a announcer) Announce(ctx context.Context) error {
	hb, err := ww.NewHeartbeat(peer.AddrInfo{
		ID:    a.host.ID(),
		Addrs: a.host.Addrs(),
	}, a.ttl)
	if err != nil {
		return err
	}

	b, err := ww.MarshalHeartbeat(hb)
	if err != nil {
		return err
	}

	return a.Publish(ctx, a.ns, b)
}

func (a announcer) Denounce(ctx context.Context) error {
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(make([]byte, 0, 16)))
	if err != nil {
		return err
	}

	goaway, err := api.NewRootGoAway(seg)
	if err != nil {
		return err
	}

	if err := goaway.SetId(string(a.host.ID())); err != nil {
		return err
	}

	b, err := msg.MarshalPacked()
	if err != nil {
		return err
	}

	return a.Publish(ctx, fmt.Sprintf("%s.leave", a.ns), b)
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
		Source: rand.New(randutil.FromPeer(a.host.ID())),
	})
	defer ticker.Stop()

	for range ticker.C {
		switch err := a.Announce(ctx); err {
		case nil, context.Canceled:
		default:
			// N.B.:  this might also happen if the topic was somehow closed.
			panic(errors.Wrap(err, "failed to publish (is pubsub enabled?)"))
		}
	}
}

type publisher interface {
	Publish(context.Context, string, []byte) error
}

type hostinfo interface {
	ID() peer.ID
	Addrs() []multiaddr.Multiaddr
}
