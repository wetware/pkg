package routing

import (
	"time"

	capnp "zombiezen.com/go/capnproto2"

	"github.com/pkg/errors"

	"github.com/wetware/ww/internal/api"
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

	return Heartbeat{hb: hb}, nil
}

// Heartbeat is a message that announces a host's liveliness in a cluster.
type Heartbeat struct {
	hb api.Heartbeat
}

// NewHeartbeat message.
func NewHeartbeat(ttl time.Duration) (Heartbeat, error) {
	_, seg, err := capnp.NewMessage(capnp.SingleSegment(make([]byte, 0, 64)))
	if err != nil {
		return Heartbeat{}, errors.Wrap(err, "new message")
	}

	hb, err := api.NewRootHeartbeat(seg)
	if err != nil {
		return Heartbeat{}, errors.Wrap(err, "root heartbeat")
	}

	hb.SetTtl(int64(ttl))

	return Heartbeat{hb: hb}, nil
}

// TTL is the duration after which the peer should be considered stale if no further
// heartbeats have been received.
func (h Heartbeat) TTL() time.Duration {
	return time.Duration(h.hb.Ttl())
}
