package client

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"

	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"

	ww "github.com/lthibault/wetware/pkg"
	"github.com/lthibault/wetware/pkg/boot"
)

var runtime = fx.Invoke(
	subloop,
	join,
)

func subloop(ctx context.Context, h host.Host, t *pubsub.Topic) fx.Hook {
	var sub *pubsub.Subscription

	return fx.Hook{
		OnStart: func(context.Context) (err error) {
			if sub, err = t.Subscribe(); err == nil {
				go recvHeartbeats(ctx, sub, h.Peerstore())
			}

			return
		},
		OnStop: func(context.Context) error {
			sub.Cancel()
			return nil
		},
	}
}

func recvHeartbeats(ctx context.Context, sub *pubsub.Subscription, a peerstore.AddrBook) {
	for {
		msg, err := sub.Next(ctx)
		if err != nil {
			break // can only be cancelled context or closed subscription
		}

		hb := msg.ValidatorData.(ww.Heartbeat)

		as := hb.Addrs()
		addrs := make([]multiaddr.Multiaddr, as.Len()) // TODO:  sync.Pool? benchmark first.

		var b []byte
		for i := 0; i < as.Len(); i++ {
			b, _ = as.At(i) // already validated; err is nil.
			addrs[i], _ = multiaddr.NewMultiaddrBytes(b)
		}

		// TODO:  default Peerstore GC is on the order of hours.  GC algoritm seems
		// 		  inefficient.  We can probably do better with a heap-filter.
		a.AddAddrs(hb.ID(), addrs, hb.TTL())
	}
}

func join(lx fx.Lifecycle, host host.Host, b boot.Strategy) {
	lx.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ps, err := b.DiscoverPeers(ctx)
			if err != nil {
				return errors.Wrap(err, "discover")
			}

			// TODO:  change this to an at-least-one-succeeds group
			var g errgroup.Group
			for _, pinfo := range ps {
				g.Go(connect(ctx, host, pinfo))
			}

			return errors.Wrap(g.Wait(), "join")
		},
	})
}

func connect(ctx context.Context, host host.Host, pinfo peer.AddrInfo) func() error {
	return func() error {
		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		return host.Connect(ctx, pinfo)
	}
}

func seqno(msg *pubsub.Message) uint64 {
	return binary.BigEndian.Uint64(msg.GetSeqno())
}
