package server

import (
	"context"
	"encoding/binary"

	"go.uber.org/fx"

	peer "github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	ww "github.com/lthibault/wetware/pkg"
	"github.com/lthibault/wetware/pkg/internal/filter"
)

type routerConfig struct {
	fx.In

	Ctx context.Context
	NS  string `name:"ns"`

	PubSub *pubsub.PubSub
}

// newRouter provides a continuous snapshot of live hosts in a cluster.
// It returns a RoutingTable along with the Topic to which it is subscribed.
func newRouter(lx fx.Lifecycle, cfg routerConfig) (*router, error) {
	r := &router{
		ctx: cfg.Ctx,
		f:   filter.New(),
	}

	err := cfg.PubSub.RegisterTopicValidator(cfg.NS, newHeartbeatValidator(cfg.Ctx, r.f))
	if err != nil {
		return nil, err
	}

	if r.t, err = cfg.PubSub.Join(cfg.NS); err != nil {
		return nil, err
	}

	// we need to consume messages for pubsub validator (i.e.: routing table) to work.
	if r.sub, err = r.t.Subscribe(); err == nil {
		go func() {
			for {
				if _, err = r.sub.Next(cfg.Ctx); err != nil {
					break
				}
			}
		}()
	}

	if err == nil {
		lx.Append(fx.Hook{
			OnStop: func(context.Context) error {
				return r.Close()
			},
		})
	}

	return r, err
}

// router tracks liveliness of peers in a cluster.
type router struct {
	f filter.Filter
	t *pubsub.Topic

	ctx context.Context
	sub *pubsub.Subscription
}

// Close the cluster topic and stop the router.
func (r router) Close() error {
	r.sub.Cancel()
	return r.t.Close()
}

// Peers returns a list all currently reachable peers.
func (r router) Peers() peer.IDSlice {
	return r.f.Peers()
}

// Topic for cluster announcements.
func (r router) Topic() *pubsub.Topic {
	return r.t
}

func newHeartbeatValidator(ctx context.Context, f filter.Filter) pubsub.Validator {
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
