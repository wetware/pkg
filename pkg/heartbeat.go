package ww

import (
	"errors"
	"time"

	capnp "zombiezen.com/go/capnproto2"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/lthibault/wetware/internal/api"
)

// MarshalHeartbeat serializes a heartbeat message.
func MarshalHeartbeat(h Heartbeat) ([]byte, error) {
	return h.msg.MarshalPacked()
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

	return Heartbeat{msg: msg, hb: hb}, validateHeartbeat(hb)
}

// Heartbeat is a message that announces a host's liveliness in a cluster.
type Heartbeat struct {
	msg *capnp.Message
	hb  api.Heartbeat
}

// NewHeartbeat message.
func NewHeartbeat(info peer.AddrInfo, ttl time.Duration) (Heartbeat, error) {
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(make([]byte, 0, 64)))
	if err != nil {
		return Heartbeat{}, err
	}

	hb, err := api.NewRootHeartbeat(seg)
	if err != nil {
		return Heartbeat{}, err
	}

	hb.SetTtl(int64(ttl))

	if err = hb.SetId(string(info.ID)); err != nil {
		return Heartbeat{}, err
	}

	as, err := hb.NewAddrs(int32(len(info.Addrs)))
	if err != nil {
		return Heartbeat{}, err
	}

	for i, a := range info.Addrs {
		if err = as.Set(i, a.Bytes()); err != nil {
			return Heartbeat{}, err
		}
	}

	return Heartbeat{msg: msg, hb: hb}, nil
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

// Addrs on which the peer is listening for connections.
func (h Heartbeat) Addrs() ([]multiaddr.Multiaddr, error) {
	as, err := h.hb.Addrs()
	if err != nil {
		return nil, err
	}

	var b []byte
	addrs := make([]multiaddr.Multiaddr, as.Len())
	for i := 0; i < as.Len(); i++ {
		if b, err = as.At(i); err != nil {
			break
		}

		if addrs[i], err = multiaddr.NewMultiaddrBytes(b); err != nil {
			break
		}
	}

	return addrs, err
}

// // ToEvent converts the heartbeat into a local event.
// func (h Heartbeat) ToEvent() (EvtHeartbeat, error) {
// 	addrs, err := h.Addrs()
// 	return EvtHeartbeat{
// 		ID:    h.ID(),
// 		TTL:   h.TTL(),
// 		Addrs: addrs,
// 	}, err
// }

func validateHeartbeat(hb api.Heartbeat) error {
	for _, test := range []struct {
		Call func() bool
		Err  string
	}{{
		Call: hb.HasId,
		Err:  "missing peer ID",
	}, {
		Call: hb.HasAddrs,
		Err:  "missing multiaddrs",
	}} {
		if !test.Call() {
			return errors.New(test.Err)
		}
	}

	return nil
}
