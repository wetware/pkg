package ww

import (
	"time"

	capnp "zombiezen.com/go/capnproto2"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"

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
func (h Heartbeat) Addrs() capnp.DataList {
	as, err := h.hb.Addrs()
	if err != nil {
		panic(err) // already validated
	}

	return as
}

func validateHeartbeat(hb api.Heartbeat) error {
	if !hb.HasId() {
		return errors.New("missing peer ID")
	}

	if !hb.HasAddrs() {
		return errors.New("missing addrs")
	}

	as, err := hb.Addrs()
	if err != nil {
		return errors.Wrap(err, "addrs datalist")
	}

	var b []byte
	for i := 0; i < as.Len(); i++ {
		if b, err = as.At(i); err != nil {
			return errors.Wrapf(err, "at datalist index %d", i)
		}

		if err = validateMultiaddrBytes(b); err != nil {
			return errors.Wrapf(err, "invalid multiaddr at index %d", i)
		}
	}

	return nil
}

/*
	Lifted from multiaddr package.
*/

func validateMultiaddrBytes(b []byte) (err error) {
	if len(b) == 0 {
		return errors.New("empty multiaddr")
	}
	for len(b) > 0 {
		code, n, err := multiaddr.ReadVarintCode(b)
		if err != nil {
			return err
		}

		b = b[n:]
		p := multiaddr.ProtocolWithCode(code)
		if p.Code == 0 {
			return errors.Errorf("no protocol with code %d", code)
		}

		if p.Size == 0 {
			continue
		}

		n, size, err := sizeForAddr(p, b)
		if err != nil {
			return err
		}

		b = b[n:]

		if len(b) < size || size < 0 {
			return errors.Errorf("invalid value for size %d", len(b))
		}

		err = p.Transcoder.ValidateBytes(b[:size])
		if err != nil {
			return err
		}

		b = b[size:]
	}

	return nil
}

func sizeForAddr(p multiaddr.Protocol, b []byte) (skip, size int, err error) {
	switch {
	case p.Size > 0:
		return 0, (p.Size / 8), nil
	case p.Size == 0:
		return 0, 0, nil
	default:
		size, n, err := multiaddr.ReadVarintCode(b)
		if err != nil {
			return 0, 0, err
		}
		return n, size, nil
	}
}
