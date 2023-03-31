package service

import (
	"fmt"
	"path"

	"capnproto.org/go/capnp/v3"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/wetware/ww/internal/api/cluster"
	api "github.com/wetware/ww/internal/api/service"
	"github.com/wetware/ww/pkg/anchor"
)

type RecordType = api.Record_Which

const (
	EnvelopeDomain = "casm-boot-record"

	TypeSturdyRef    = api.Record_Which_sturdyRef
	TypeMultiaddr    = api.Record_Which_multiaddr
	TypeAnchor       = api.Record_Which_anchor
	TypeCustomStruct = api.Record_Which_customStruct
	TypeCustomList   = api.Record_Which_customList
	TypeCustomText   = api.Record_Which_customText
	TypeCustomData   = api.Record_Which_customData
)

type Record struct{ api.Record }

func (r *Record) Reset(arena capnp.Arena) error {
	if r.Message() != nil {
		return r.Reset(arena)
	}

	_, seg, err := capnp.NewMessage(arena)
	if err == nil {
		r.Record, err = api.NewRootRecord(seg)
	}

	return err
}

func (r Record) Type() RecordType {
	return r.Record.Which()
}

func (r *Record) ReadMessage(m *capnp.Message) (err error) {
	r.Record, err = api.ReadRootRecord(m)
	return
}

func (r Record) SturdyRef() (cluster.SturdyRef, error) { // TODO:  exported SturdyRef type
	return r.Record.SturdyRef()
}

func (r Record) SetSturdyRef(ref cluster.SturdyRef) error {
	return r.Record.SetSturdyRef(ref)
}

func (r Record) NewSturdyRef() (cluster.SturdyRef, error) {
	return r.Record.NewSturdyRef()
}

func (r Record) Multiaddr() (ma.Multiaddr, error) {
	b, err := r.Record.Multiaddr()
	if err != nil {
		return nil, err
	}

	return ma.NewMultiaddrBytes(b)
}

func (r Record) SetMultiaddr(maddr ma.Multiaddr) error {
	return r.Record.SetMultiaddr(maddr.Bytes())
}

func (r Record) SetMultiaddrString(maddr string) error {
	m, err := ma.NewMultiaddr(maddr)
	if err == nil {
		err = r.Record.SetMultiaddr(m.Bytes())
	}

	return err
}

func (r Record) AnchorPath() anchor.Path {
	path, err := r.Record.Anchor()
	if err == nil {
		return anchor.NewPath(path)
	}

	return anchor.FailPath(fmt.Errorf("%s: %w", r.Domain(), err))
}

func (r Record) SetAnchorPath(p anchor.Path) error {
	path, err := p.Unwrap()
	if err == nil {
		err = r.Record.SetAnchor(path)
	}

	return err
}

func (r Record) SetAnchorPathString(path string) error {
	return r.SetAnchorPath(anchor.NewPath(path))
}

// Domain is the "signature domain" for the record.  It is always equal to the service
// name.  This means that records obtained from pubsub.Topic T cannot be used in topic
// T', which in turn prevents cross-domain forgeries.
func (r Record) Domain() (s string) {
	if r.Record.HasServiceName() {
		s, _ = r.Record.ServiceName()
		s = path.Join("ww-service", s)
	}

	return
}

// Codec is a binary identifier for this type of record.
// When a Record is put into an Envelope (see record.Seal), the Codec value will be used
// as the Envelope's PayloadType. When the Envelope is later unsealed, the PayloadType
// will be used to lookup the correct Record type to unmarshal the Envelope payload into.
//
// TODO:  register 0x0915 to https://github.com/multiformats/multicodec
func (r Record) Codec() []byte {
	return []byte{0x09, 0x15}
}

// MarshalRecord returns a packed capnp encoding of the record.
func (r Record) MarshalRecord() ([]byte, error) {
	return r.Record.Message().MarshalPacked()
}

// UnmarshalRecord unmarshals a packed capnp encoding of the record.
func (r *Record) UnmarshalRecord(b []byte) error {
	m, err := capnp.UnmarshalPacked(b)
	if err == nil {
		err = r.ReadMessage(m)
	}

	return err
}
