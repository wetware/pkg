package pulse

import (
	"time"

	"capnproto.org/go/capnp/v3"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	api "github.com/wetware/pkg/api/cluster"
	"github.com/wetware/pkg/cluster/routing"
)

const DefaultTTL = time.Second * 10

type Heartbeat struct{ api.Heartbeat }

func NewHeartbeat() Heartbeat {
	_, seg := capnp.NewSingleSegmentMessage(nil)
	h, _ := api.NewRootHeartbeat(seg) // single segment never fails
	return Heartbeat{h}
}

func (h Heartbeat) SetTTL(d time.Duration) {
	ms := d / time.Millisecond
	h.SetTtl(uint32(ms))
}

func (h Heartbeat) TTL() (d time.Duration) {
	if d = time.Millisecond * time.Duration(h.Ttl()); d == 0 {
		d = DefaultTTL
	}

	return
}

func (h Heartbeat) SetServer(id routing.ID) {
	h.Heartbeat.SetServer(uint64(id))
}

func (h Heartbeat) Server() routing.ID {
	return routing.ID(h.Heartbeat.Server())
}

func (h Heartbeat) Meta() (routing.Meta, error) {
	meta, err := h.Heartbeat.Meta()
	return routing.Meta(meta), err
}

func (h Heartbeat) SetMeta(fields []routing.MetaField) error {
	meta, err := h.NewMeta(int32(len(fields)))
	if err != nil {
		return err
	}

	for i, f := range fields {
		if err = meta.Set(i, f.String()); err != nil {
			break
		}
	}

	return err
}

func (h *Heartbeat) ReadMessage(m *capnp.Message) (err error) {
	h.Heartbeat, err = api.ReadRootHeartbeat(m)
	return
}

func (h *Heartbeat) Bind(msg *pubsub.Message) (err error) {
	var m *capnp.Message
	if m, err = capnp.UnmarshalPacked(msg.Data); err == nil {
		msg.ValidatorData = h
		err = h.ReadMessage(m)
	}

	return
}
