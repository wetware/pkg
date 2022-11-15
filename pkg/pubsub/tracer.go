package pubsub

import (
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/lthibault/log"
)

var _ pubsub.RawTracer = (*Tracer)(nil)

type Tracer struct {
	Log     log.Logger
	Metrics interface {
		Incr(string)
		Decr(string)
	}
}

// AddPeer is invoked when a new peer is added.
func (t Tracer) AddPeer(id peer.ID, proto protocol.ID) {
	t.Log.
		WithField("peer", id).
		WithField("proto", proto).
		Trace("peer added")
	t.Metrics.Incr("peers")
}

// RemovePeer is invoked when a peer is removed.
func (t Tracer) RemovePeer(id peer.ID) {
	t.Log.
		WithField("peer", id).
		Trace("peer removed")
	t.Metrics.Decr("peers")
}

// Join is invoked when a new topic is joined
func (t Tracer) Join(topic string) {
	t.Log.
		WithField("topic", topic).
		Trace("joined topic")
	t.Metrics.Incr("topics")
}

// Leave is invoked when a topic is abandoned
func (t Tracer) Leave(topic string) {
	t.Log.
		WithField("topic", topic).
		Trace("left topic")
	t.Metrics.Decr("topics")
}

// Graft is invoked when a new peer is grafted on the mesh (gossipsub)
func (t Tracer) Graft(id peer.ID, topic string) {
	t.Log.
		WithField("peer", id).
		WithField("topic", topic).
		Trace("grafted peer")
	t.Metrics.Incr("graft")
}

// Prune is invoked when a peer is pruned from the message (gossipsub)
func (t Tracer) Prune(id peer.ID, topic string) {
	t.Log.
		WithField("peer", id).
		WithField("topic", topic).
		Trace("pruned peer")
	t.Metrics.Incr("prune")
}

// ValidateMessage is invoked when a message first enters the validation pipeline.
func (t Tracer) ValidateMessage(*pubsub.Message) {}

// DeliverMessage is invoked when a message is delivered
func (t Tracer) DeliverMessage(*pubsub.Message) {
	t.Metrics.Incr("delivered")
}

// RejectMessage is invoked when a message is Rejected or Ignored.
// The reason argument can be one of the named strings Reject*.
func (t Tracer) RejectMessage(m *pubsub.Message, reason string) {
	t.Log.
		WithField("topic", m.GetTopic()).
		WithField("reason", reason).
		Info("message rejected")
	t.Metrics.Incr("rejected")
}

// DuplicateMessage is invoked when a duplicate message is dropped.
func (t Tracer) DuplicateMessage(*pubsub.Message) {
	t.Metrics.Incr("duplicates")
}

// ThrottlePeer is invoked when a peer is throttled by the peer gater.
func (t Tracer) ThrottlePeer(id peer.ID) {
	t.Log.
		WithField("peer", id).
		Info("peer throttled")
	t.Metrics.Incr("throttled")
}

// RecvRPC is invoked when an incoming RPC is received.
func (t Tracer) RecvRPC(*pubsub.RPC) {
	t.Metrics.Incr("rpc.recved")
}

// SendRPC is invoked when a RPC is sent.
func (t Tracer) SendRPC(*pubsub.RPC, peer.ID) {
	t.Metrics.Incr("rpc.sent")
}

// DropRPC is invoked when an outbound RPC is dropped, typically because of a queue full.
func (t Tracer) DropRPC(r *pubsub.RPC, id peer.ID) {
	t.Log.
		With(rpcFields(r)).
		WithField("peer", id).
		Warn("outbound rpc failed")
	t.Metrics.Incr("rpc.dropped")
}

// UndeliverableMessage is invoked when the consumer of Subscribe is not reading messages fast enough and
// the pressure release mechanism trigger, dropping messages.
func (t Tracer) UndeliverableMessage(m *pubsub.Message) {
	// t.Log.
	// 	WithField("topic", m.GetTopic()).
	// 	Warn("message undeliverable")
	t.Metrics.Incr("undeliverable")
}

func rpcFields(r *pubsub.RPC) log.F {
	if r.Publish != nil {
		return publishFields(r)
	}

	return controlFields(r)
}

func publishFields(r *pubsub.RPC) log.F {
	return log.F{
		"rpc": "publish",
	}
}

func controlFields(r *pubsub.RPC) log.F {
	switch {
	case r.Control.Graft != nil:
		return log.F{
			"rpc":    "control",
			"method": "graft",
			"topics": graftTopics(r),
		}

	case r.Control.Prune != nil:
		return log.F{
			"rpc":    "control",
			"method": "prune",
			"topics": pruneTopics(r),
		}

	case r.Control.Iwant != nil:
		return log.F{
			"rpc":    "control",
			"method": "iwant",
		}

	case r.Control.Ihave != nil:
		return log.F{
			"rpc":    "control",
			"method": "ihave",
		}

	default:
		return log.F{
			"rpc":    "control",
			"method": "unknown",
		}
	}
}

func graftTopics(r *pubsub.RPC) []string {
	ss := make([]string, len(r.Control.Graft))
	for i, topic := range r.Control.Graft {
		ss[i] = *topic.TopicID
	}
	return ss
}

func pruneTopics(r *pubsub.RPC) []string {
	ss := make([]string, len(r.Control.Prune))
	for i, topic := range r.Control.Prune {
		ss[i] = *topic.TopicID
	}
	return ss
}
