package cluster

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"go.uber.org/fx"
)

type announcer struct{ t *pubsub.Topic }

func (a announcer) Namespace() string { return a.t.String() }

func (a announcer) Announce(ctx context.Context, ttl time.Duration) error {
	b := make([]byte, binary.MaxVarintLen64)
	return a.t.Publish(ctx, b[:binary.PutUvarint(b, uint64(ttl))])
}

func newHeartbeatValidator(f *filter) pubsub.Validator {
	return func(_ context.Context, pid peer.ID, msg *pubsub.Message) (ok bool) {
		if id := msg.GetFrom(); f.Upsert(id, seqno(msg), ttl(msg)) {
			ok = true // continue processing the message
			msg.ValidatorData = ttl(msg)
		}

		return
	}
}

// we need to consume messages for pubsub validator (i.e.: routing table) to work.
func consumer(ctx context.Context, t *pubsub.Topic) fx.Hook {
	var sub *pubsub.Subscription
	return fx.Hook{
		OnStart: func(ctx context.Context) (err error) {
			if sub, err = t.Subscribe(); err == nil {
				go func() {
					for {
						_, err := sub.Next(context.Background())
						if err != nil {
							break
						}
					}
				}()
			}

			return
		},
		OnStop: func(context.Context) error {
			sub.Cancel()
			return t.Close()
		},
	}
}

func seqno(msg *pubsub.Message) uint64 {
	return binary.BigEndian.Uint64(msg.GetSeqno())
}

func ttl(msg *pubsub.Message) time.Duration {
	d, _ := binary.Varint(msg.GetData())
	return time.Duration(d)
}
