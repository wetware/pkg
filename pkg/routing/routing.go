package routing

import (
	"context"
	"encoding/binary"

	log "github.com/lthibault/log/pkg"
	"go.uber.org/fx"

	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// Table provides a snapshot of active hosts in a cluster.
type Table interface {
	Peers() peer.IDSlice
}

// Config for router
type Config struct {
	fx.In

	Ctx context.Context
	Log log.Logger

	Namespace string `name:"ns"`

	Host   host.Host
	PubSub *pubsub.PubSub
}

// Module containing router primitives
type Module struct {
	fx.Out

	Table Table
	Topic *pubsub.Topic
}

// New router, which provides a continuous snapshot of live hosts in a cluster.
// It returns a Table along with the Topic to which it is subscribed.
func New(ctx context.Context, cfg Config, lx fx.Lifecycle) (mod Module, err error) {
	f := newFilter()
	mod.Table = f

	validator := newHeartbeatValidator(ctx, f)
	if err = cfg.PubSub.RegisterTopicValidator(cfg.Namespace, validator); err != nil {
		return
	}

	if mod.Topic, err = cfg.PubSub.Join(cfg.Namespace); err != nil {
		return
	}
	lx.Append(consumerhook(ctx, mod.Topic))

	return
}

func newHeartbeatValidator(ctx context.Context, f *filter) pubsub.Validator {
	return func(_ context.Context, pid peer.ID, msg *pubsub.Message) bool {
		hb, err := UnmarshalHeartbeat(msg.GetData())
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

// we need to consume messages for pubsub validator (i.e.: routing table) to work.
func consumerhook(ctx context.Context, t *pubsub.Topic) fx.Hook {
	var sub *pubsub.Subscription
	return fx.Hook{
		OnStart: func(context.Context) (err error) {
			if sub, err = t.Subscribe(); err == nil {
				go consume(ctx, sub)
			}

			return
		},
		OnStop: func(context.Context) error {
			sub.Cancel()
			return t.Close()
		},
	}
}

func consume(ctx context.Context, sub *pubsub.Subscription) {
	for _, err := sub.Next(ctx); err != nil; {
		// busy loop
	}
}

func seqno(msg *pubsub.Message) uint64 {
	return binary.BigEndian.Uint64(msg.GetSeqno())
}
