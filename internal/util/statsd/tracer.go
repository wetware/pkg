package statsdutil

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/urfave/cli/v2"
	"gopkg.in/alexcesaro/statsd.v2"
)

type PubSubTracer statsd.Client

func NewPubSubTracer(c *cli.Context) *PubSubTracer {
	return (*PubSubTracer)(Must(c).Clone(
		statsd.Prefix("pubsub.")))
}

// AddPeer is invoked when a new peer is added.
func (t *PubSubTracer) AddPeer(peer.ID, protocol.ID) {
	(*statsd.Client)(t).Increment("peers")
}

// RemovePeer is invoked when a peer is removed.
func (t *PubSubTracer) RemovePeer(p peer.ID) {
	(*statsd.Client)(t).Count("peers", -1)
}

// Join is invoked when a new topic is joined
func (t *PubSubTracer) Join(topic string) {
	(*statsd.Client)(t).Increment("topics")
}

// Leave is invoked when a topic is abandoned
func (t *PubSubTracer) Leave(topic string) {
	(*statsd.Client)(t).Count("topics", -1)
}

// Graft is invoked when a new peer is grafted on the mesh (gossipsub)
func (t *PubSubTracer) Graft(peer.ID, string) {
	(*statsd.Client)(t).Increment("graft")
}

// Prune is invoked when a peer is pruned from the message (gossipsub)
func (t *PubSubTracer) Prune(peer.ID, string) {
	(*statsd.Client)(t).Increment("prune")
}

// ValidateMessage is invoked when a message first enters the validation pipeline.
func (t *PubSubTracer) ValidateMessage(*pubsub.Message) {}

// DeliverMessage is invoked when a message is delivered
func (t *PubSubTracer) DeliverMessage(*pubsub.Message) {
	(*statsd.Client)(t).Increment("delivered")
}

// RejectMessage is invoked when a message is Rejected or Ignored.
// The reason argument can be one of the named strings Reject*.
func (t *PubSubTracer) RejectMessage(*pubsub.Message, string) {
	(*statsd.Client)(t).Increment("rejected")
}

// DuplicateMessage is invoked when a duplicate message is dropped.
func (t *PubSubTracer) DuplicateMessage(msg *pubsub.Message) {
	(*statsd.Client)(t).Increment("duplicates")
}

// ThrottlePeer is invoked when a peer is throttled by the peer gater.
func (t *PubSubTracer) ThrottlePeer(p peer.ID) {
	(*statsd.Client)(t).Increment("throttled")
}

// RecvRPC is invoked when an incoming RPC is received.
func (t *PubSubTracer) RecvRPC(*pubsub.RPC) {
	(*statsd.Client)(t).Increment("rpc.recved")
}

// SendRPC is invoked when a RPC is sent.
func (t *PubSubTracer) SendRPC(*pubsub.RPC, peer.ID) {
	(*statsd.Client)(t).Increment("rpc.sent")
}

// DropRPC is invoked when an outbound RPC is dropped, typically because of a queue full.
func (t *PubSubTracer) DropRPC(*pubsub.RPC, peer.ID) {
	(*statsd.Client)(t).Increment("rpc.dropped")
}

// UndeliverableMessage is invoked when the consumer of Subscribe is not reading messages fast enough and
// the pressure release mechanism trigger, dropping messages.
func (t *PubSubTracer) UndeliverableMessage(msg *pubsub.Message) {
	(*statsd.Client)(t).Increment("undeliverable")
}
