//go:generate mockgen -source=pulse.go -destination=test/pulse.go -package=test_pulse

// Package pulse provides ev cluster-management service based on pubsub.
package pulse

import (
	"context"
	"encoding/binary"
	"unsafe"

	"capnproto.org/go/capnp/v3"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	b58 "github.com/mr-tron/base58/base58"

	api "github.com/wetware/pkg/api/cluster"
	"github.com/wetware/pkg/cluster/routing"
)

type RoutingTable interface {
	Upsert(routing.Record) bool
}

type Preparer interface {
	Prepare(Heartbeat) error
}

func NewValidator(rt RoutingTable) pubsub.ValidatorEx {
	return func(_ context.Context, _ peer.ID, m *pubsub.Message) pubsub.ValidationResult {
		if rec, err := record(m); err == nil {
			if rt.Upsert(rec) {
				return pubsub.ValidationAccept
			}

			// heartbeat is valid, but we have a more recent one.
			return pubsub.ValidationIgnore
		}

		// assume the worst...
		return pubsub.ValidationReject
	}
}

type routingRecord struct {
	peer, seq []byte
	Heartbeat
}

func record(msg *pubsub.Message) (*routingRecord, error) {
	rec := &routingRecord{
		peer: msg.Message.GetFrom(),
		seq:  msg.GetSeqno(),
	}

	m, err := capnp.UnmarshalPacked(msg.GetData())
	if err != nil {
		return nil, err
	}

	return rec, rec.ReadMessage(m)
}

func (r *routingRecord) Peer() peer.ID {
	return *(*peer.ID)(unsafe.Pointer(&r.peer))
}

func (r *routingRecord) PeerBytes() ([]byte, error) {
	id := b58.Encode(r.peer)
	return *(*[]byte)(unsafe.Pointer(&id)), nil
}

func (r *routingRecord) Seq() uint64 {
	return binary.BigEndian.Uint64(r.seq)
}

func (r *routingRecord) BindRecord(rec api.View_Record) (err error) {
	if err = rec.SetPeer(string(r.Peer())); err == nil {
		rec.SetSeq(r.Seq())
		err = rec.SetHeartbeat(r.Heartbeat.Heartbeat)
	}

	return
}
