package routing

import (
	"time"

	capnp "zombiezen.com/go/capnproto2"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"

	"github.com/lthibault/wetware/internal/api"
)

// MarshalHeartbeat serializes a heartbeat message.
func MarshalHeartbeat(h Heartbeat) ([]byte, error) {
	return h.hb.Segment().Message().MarshalPacked()
}

// UnmarshalHeartbeat reads a heartbeat message from bytes.
func UnmarshalHeartbeat(b []byte) (Heartbeat, error) {
	msg, err := capnp.UnmarshalPacked(b)
	if err != nil {
		return Heartbeat{}, err
	}

	hb, err := api.ReadRootHeartbeat(msg)
	if err != nil {
		return Heartbeat{}, err
	}

	return Heartbeat{hb: hb}, validateHeartbeat(hb)
}

// Heartbeat is a message that announces a host's liveliness in a cluster.
type Heartbeat struct {
	hb api.Heartbeat
}

// NewHeartbeat message.
func NewHeartbeat(id peer.ID, ttl time.Duration) (Heartbeat, error) {
	_, seg, err := capnp.NewMessage(capnp.SingleSegment(make([]byte, 0, 64)))
	if err != nil {
		return Heartbeat{}, errors.Wrap(err, "new message")
	}

	hb, err := api.NewRootHeartbeat(seg)
	if err != nil {
		return Heartbeat{}, errors.Wrap(err, "root heartbeat")
	}

	hb.SetTtl(int64(ttl))

	if err = hb.SetId(string(id)); err != nil {
		return Heartbeat{}, errors.Wrap(err, "set id")
	}

	return Heartbeat{hb: hb}, nil
}

// ID of the peer that emitted the heartbeat.
func (h Heartbeat) ID() peer.ID {
	id, err := h.hb.Id()
	if err != nil {
		panic(err) // should have been caught by validation
	}
	return peer.ID(id)
}

// TTL is the duration after which the peer should be considered stale if no further
// heartbeats have been received.
func (h Heartbeat) TTL() time.Duration {
	return time.Duration(h.hb.Ttl())
}

func validateHeartbeat(hb api.Heartbeat) error {
	if !hb.HasId() {
		return errors.New("missing peer ID")
	}

	return nil
}
