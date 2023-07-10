package service

import (
	"errors"
	"fmt"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p-core/record"
	ma "github.com/multiformats/go-multiaddr"
	api "github.com/wetware/ww/api/registry"
)

// TODO:  register this once stable.
// https://github.com/multiformats/multicodec/blob/master/table.csv

var (
	EnvelopeDomain      = "ww/registry/location"
	EnvelopePayloadType = []byte{0x1f, 0x01}
	ErrInavlidType      = errors.New("invalid type")
)

func init() {
	record.RegisterType(&Location{})
}

var ErrInvalidSignature = errors.New("invalid signature")

type Location struct {
	api.Location
}

func NewLocation() (Location, error) {
	_, seg := capnp.NewSingleSegmentMessage(nil)
	loc, err := api.NewRootLocation(seg)
	if err != nil {
		return Location{}, fmt.Errorf("failed to create location: %w", err)
	}

	return Location{Location: loc}, nil
}

func (loc Location) Validate(topic string) error {
	service, err := loc.Service()
	if err != nil {
		return fmt.Errorf("failed to read service name: %w", err)
	}
	if topic != service {
		return fmt.Errorf("the topic and the service name are different, topic: %s - service: %s", topic, service)
	}

	return nil
}

func (loc Location) SetMaddrs(maddrs []ma.Multiaddr) error {
	capMaddrs, err := loc.NewMaddrs(int32(len(maddrs)))
	if err != nil {
		return fmt.Errorf("fail to create capnp Multiaddr: %w", err)
	}

	for i, maddr := range maddrs {
		if err := capMaddrs.Set(i, maddr.Bytes()); err != nil {
			return fmt.Errorf("fail to set maddr in Location: %w", err)
		}
	}

	return nil
}

func (loc Location) SetAnchor(anchor string) error {
	if err := loc.SetAnchor(anchor); err != nil {
		return fmt.Errorf("fail to set anchor in capnp Location: %w", err)
	}
	return nil
}

func (loc Location) SetCustom(custom capnp.Ptr) error {
	if err := loc.SetCustom(custom); err != nil {
		return fmt.Errorf("fail to set custom in capnp Location: %w", err)
	}
	return nil
}

func (loc Location) Maddrs() ([]ma.Multiaddr, error) {
	capMaddrs, err := loc.Location.Maddrs()
	if err != nil {
		return nil, fmt.Errorf("failed to get Multiaddresses: %w", err)
	}

	maddrs := make([]ma.Multiaddr, 0, capMaddrs.Len())
	for i := 0; i < capMaddrs.Len(); i++ {
		buffer, err := capMaddrs.At(i)
		if err != nil {
			return nil, fmt.Errorf("failed to get Multiaddress at index %d: %w", i, err)
		}

		b := make([]byte, len(buffer))
		copy(b, buffer)

		maddr, err := ma.NewMultiaddrBytes(b)
		if err != nil {
			return nil, fmt.Errorf("failed to decode Multiaddress: %w", err)
		}
		maddrs = append(maddrs, maddr)
	}
	return maddrs, nil
}

// Domain is the "signature domain" used when signing and verifying a particular
// Record type. The Domain string should be unique to your Record type, and all
// instances of the Record type must have the same Domain string.
func (loc Location) Domain() string {
	return EnvelopeDomain
}

// Codec is a binary identifier for this type of record, ideally a registered multicodec
// (see https://github.com/multiformats/multicodec).
// When a Record is put into an Envelope (see record.Seal), the Codec value will be used
// as the Envelope's PayloadType. When the Envelope is later unsealed, the PayloadType
// will be used to lookup the correct Record type to unmarshal the Envelope payload into.
func (loc Location) Codec() []byte {
	return EnvelopePayloadType
}

// MarshalRecord converts a Record instance to a []byte, so that it can be used as an
// Envelope payload.
func (loc Location) MarshalRecord() ([]byte, error) {
	return loc.Message().MarshalPacked()
}

// UnmarshalRecord unmarshals a []byte payload into an instance of a particular Record type.
func (loc *Location) UnmarshalRecord(b []byte) error {
	m, err := capnp.UnmarshalPacked(b)
	if err != nil {
		return err
	}

	capLoc, err := api.ReadRootLocation(m)
	if err != nil {
		return err
	}

	loc.Location = capLoc

	return nil
}
