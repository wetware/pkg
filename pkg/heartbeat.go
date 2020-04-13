package ww

import (
	"errors"
	"time"

	capnp "zombiezen.com/go/capnproto2"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/lthibault/wetware/internal/api"
)

type heartbeat struct {
	msg *capnp.Message
	hb  api.Heartbeat
}

func newHeartbeat(host listenProvider, ttl time.Duration) (heartbeat, error) {
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(make([]byte, 0, 64)))
	if err != nil {
		return heartbeat{}, err
	}

	hb, err := api.NewRootHeartbeat(seg)
	if err != nil {
		return heartbeat{}, err
	}

	hb.SetTtl(int64(ttl))

	if err = hb.SetId(string(host.ID())); err != nil {
		return heartbeat{}, err
	}

	addrs := host.Addrs()
	as, err := hb.NewAddrs(int32(len(addrs)))
	if err != nil {
		return heartbeat{}, err
	}

	for i, a := range addrs {
		if err = as.Set(i, a.Bytes()); err != nil {
			return heartbeat{}, err
		}
	}

	return heartbeat{msg: msg, hb: hb}, nil
}

func (h heartbeat) ID() peer.ID {
	id, err := h.hb.Id()
	if err != nil {
		panic(err) // should have been caught by validation
	}
	return peer.ID(id)
}

func (h heartbeat) TTL() time.Duration {
	return time.Duration(h.hb.Ttl())
}

func (h heartbeat) Addrs() ([]multiaddr.Multiaddr, error) {
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

func (h heartbeat) ToEvent() (EvtHeartbeat, error) {
	addrs, err := h.Addrs()
	return EvtHeartbeat{
		ID:    h.ID(),
		TTL:   h.TTL(),
		Addrs: addrs,
	}, err
}

func unmarshalHeartbeat(b []byte) (heartbeat, error) {
	msg, err := capnp.UnmarshalPacked(b)
	if err != nil {
		return heartbeat{}, err
	}

	hb, err := api.ReadRootHeartbeat(msg)
	if err != nil {
		return heartbeat{}, err
	}

	return heartbeat{msg: msg, hb: hb}, validateHeartbeat(hb)
}

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

func marshalHeartbeat(h heartbeat) ([]byte, error) {
	return h.msg.MarshalPacked()
}

type listenProvider interface {
	ID() peer.ID
	Addrs() []multiaddr.Multiaddr
}
