package client

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
	"go.uber.org/fx"

	ww "github.com/lthibault/wetware/pkg"
)

func newHeartbeatValidator(ctx context.Context) pubsub.Validator {
	var f shardedFilter

	// Start a background task that periodically evict stale entries from the filter
	// array.
	go func() {
		ticker := time.NewTicker(time.Millisecond * 200)
		defer ticker.Stop()

		for {
			select {
			case t := <-ticker.C:
				f.Advance(t)
			case <-ctx.Done():
				return
			}
		}
	}()

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

func subloop(ctx context.Context, t *pubsub.Topic, a peerstore.AddrBook) fx.Hook {
	var sub *pubsub.Subscription

	return fx.Hook{
		OnStart: func(context.Context) (err error) {
			if sub, err = t.Subscribe(); err == nil {
				go recvHeartbeats(ctx, sub, a)
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

func seqno(msg *pubsub.Message) uint64 {
	return binary.BigEndian.Uint64(msg.GetSeqno())
}
