package cluster

import (
	"context"
	"encoding/binary"

	peer "github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// Router tracks liveliness of peers in a cluster.
type Router struct {
	f filter
	t *pubsub.Topic

	ctx context.Context
	sub *pubsub.Subscription
}

// Close the cluster topic and stop the router.
func (r Router) Close() error {
	r.sub.Cancel()
	return r.t.Close()
}

// Peers returns a list all currently reachable peers.
func (r Router) Peers() peer.IDSlice {
	return r.f.Peers()
}

// Topic for cluster announcements.
func (r Router) Topic() *pubsub.Topic {
	return r.t
}

// NewRouter provides a continuous snapshot of live hosts in a cluster.
// It returns a RoutingTable along with the Topic to which it is subscribed.
func NewRouter(ctx context.Context, ns string, ps *pubsub.PubSub) (r *Router, err error) {
	r = &Router{
		ctx: ctx,
		f:   newBasicFilter(),
	}

	if err = ps.RegisterTopicValidator(ns, newHeartbeatValidator(ctx, r.f)); err != nil {
		return
	}

	if r.t, err = ps.Join(ns); err != nil {
		return
	}

	// we need to consume messages for pubsub validator (i.e.: routing table) to work.
	if r.sub, err = r.t.Subscribe(); err == nil {
		go func() {
			for {
				if _, err = r.sub.Next(ctx); err != nil {
					break
				}
			}
		}()
	}

	return
}

func newHeartbeatValidator(ctx context.Context, f filter) pubsub.Validator {
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

func seqno(msg *pubsub.Message) uint64 {
	return binary.BigEndian.Uint64(msg.GetSeqno())
}
