package socket

import (
	"fmt"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/record"
	"github.com/wetware/pkg/api/boot"
)

type RecordType = boot.Packet_Which

const (
	EnvelopeDomain = "casm-boot-record"

	TypeRequest  RecordType = boot.Packet_Which_request
	TypeSurvey   RecordType = boot.Packet_Which_survey
	TypeResponse RecordType = boot.Packet_Which_response
)

// TODO:  register this once stable.
// https://github.com/multiformats/multicodec/blob/master/table.csv
var EnvelopePayloadType = []byte{0x1f, 0x00}

func init() {
	record.RegisterType(&Record{})
}

type Record boot.Packet

func (r Record) Type() RecordType {
	return r.asPacket().Which()
}

func (r Record) asPacket() boot.Packet { return boot.Packet(r) }

func (r Record) Namespace() (string, error) {
	return boot.Packet(r).Namespace()
}

func (r Record) Peer() (id peer.ID, err error) {
	var b []byte
	switch r.Type() {
	case boot.Packet_Which_request:
		if b, err = r.asPacket().Request().FromBytes(); err == nil {
			id, err = peer.IDFromBytes(b)
		}

	case boot.Packet_Which_survey:
		if b, err = r.asPacket().Survey().FromBytes(); err == nil {
			id, err = peer.IDFromBytes(b)
		}

	case boot.Packet_Which_response:
		if b, err = r.asPacket().Response().PeerBytes(); err == nil {
			id, err = peer.IDFromBytes(b)
		}

	default:
		err = fmt.Errorf("unrecognized record type %s", r.Type())
	}

	return
}

// Domain is the "signature domain" used when signing and verifying a particular
// Record type. The Domain string should be unique to your Record type, and all
// instances of the Record type must have the same Domain string.
func (r Record) Domain() string { return EnvelopeDomain }

// Codec is a binary identifier for this type of record, ideally a registered multicodec
// (see https://github.com/multiformats/multicodec).
// When a Record is put into an Envelope (see record.Seal), the Codec value will be used
// as the Envelope's PayloadType. When the Envelope is later unsealed, the PayloadType
// will be used to lookup the correct Record type to unmarshal the Envelope payload into.
func (r Record) Codec() []byte { return EnvelopePayloadType }

// MarshalRecord converts a Record instance to a []byte, so that it can be used as an
// Envelope payload.
func (r Record) MarshalRecord() ([]byte, error) {
	return boot.Packet(r).Message().MarshalPacked()
}

// UnmarshalRecord unmarshals a []byte payload into an instance of a particular Record type.
func (r *Record) UnmarshalRecord(b []byte) error {
	m, err := capnp.UnmarshalPacked(b)
	if err == nil {
		*(*boot.Packet)(r), err = boot.ReadRootPacket(m)
	}

	return err
}
