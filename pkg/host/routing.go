package host

import (
	"context"
	"encoding/binary"

	"go.uber.org/fx"

	peer "github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/wetware/pkg/internal/filter"
	"github.com/lthibault/wetware/pkg/routing"
)

type routingTopicParams struct {
	fx.In

	Namespace string `name:"ns"`
	Filter    filter.Filter
	PubSub    *pubsub.PubSub
}

func routingTopic(ctx context.Context, lx fx.Lifecycle, ps routingTopicParams) (t *pubsub.Topic, err error) {
	validator := newHeartbeatValidator(ps.Filter)
	if err = ps.PubSub.RegisterTopicValidator(ps.Namespace, validator); err != nil {
		return
	}

	if t, err = ps.PubSub.Join(ps.Namespace); err != nil {
		return
	}
	lx.Append(closehook(ctx, t))

	return
}

func newHeartbeatValidator(f filter.Filter) pubsub.Validator {
	return func(_ context.Context, pid peer.ID, msg *pubsub.Message) bool {
		hb, err := routing.UnmarshalHeartbeat(msg.GetData())
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
func closehook(ctx context.Context, t *pubsub.Topic) fx.Hook {
	return fx.Hook{
		OnStop: func(context.Context) error {
			return t.Close()
		},
	}
}

func seqno(msg *pubsub.Message) uint64 {
	return binary.BigEndian.Uint64(msg.GetSeqno())
}
