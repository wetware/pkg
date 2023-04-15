package service

import (
	"errors"
	"fmt"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	api "github.com/wetware/ww/internal/api/service"
)

var ErrInvalidSignature = errors.New("invalid signature")

type Location struct {
	api.SignedLocation
}

func NewLocation() (Location, error) {
	_, seg := capnp.NewSingleSegmentMessage(nil)
	sloc, err := api.NewSignedLocation(seg)
	if err != nil {
		return Location{}, fmt.Errorf("failed to create location: %w", err)
	}

	_, seg = capnp.NewSingleSegmentMessage(nil)
	loc, err := api.NewLocation(seg)
	if err != nil {
		return Location{}, fmt.Errorf("failed to create location: %w", err)
	}

	if err := sloc.SetLocation(loc); err != nil {
		return Location{}, fmt.Errorf("failed to set location: %w", err)
	}

	return Location{SignedLocation: sloc}, nil
}

func (loc Location) Validate(topic string) error {
	capLoc, err := loc.Location()
	if err != nil {
		return fmt.Errorf("failed to read location: %w", err)
	}
	service, err := capLoc.Service()
	if err != nil {
		return fmt.Errorf("failed to read service name: %w", err)
	}
	if topic != service {
		return fmt.Errorf("the topic and the service name are different, topic: %s - service: %s", topic, service)
	}

	if err := loc.VerifySignature(); err != nil {
		return fmt.Errorf("failed to verify signature: %w", err)
	}

	return nil
}

func (loc Location) Sign(pk crypto.PrivKey) error {
	capLoc, err := loc.Location()
	if err != nil {
		return fmt.Errorf("failed to read location: %w", err)
	}

	b, err := capLoc.Message().Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal location: %w", err)
	}

	signature, err := pk.Sign(b)
	if err != nil {
		return fmt.Errorf("failed to sign location: %w", err)
	}

	if err := loc.SetSignature(signature); err != nil {
		return fmt.Errorf("failed to set signature: %w", err)
	}

	return nil
}

func (loc Location) VerifySignature() error {
	capLoc, err := loc.Location()
	if err != nil {
		return fmt.Errorf("failed to read location: %w", err)
	}

	b, err := capLoc.Message().Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal location: %w", err)
	}

	idBytes, err := capLoc.Id()
	if err != nil {
		return fmt.Errorf("failed to extract peer ID: %w", err)
	}
	peerID := peer.ID(idBytes)
	pubKey, err := peerID.ExtractPublicKey()
	if err != nil {
		return fmt.Errorf("failed to extract public key: %w", err)
	}

	sig, err := loc.Signature()
	if err != nil {
		return fmt.Errorf("failed to extract signature: %w", err)
	}

	if ok, err := pubKey.Verify(b, sig); err != nil {
		return fmt.Errorf("failed to verify signature: %w", err)
	} else if !ok {
		return ErrInvalidSignature
	}

	return nil
}

func (loc Location) SetService(name string) error {
	capLoc, err := loc.Location()
	if err != nil {
		return fmt.Errorf("fail to create location: %w", err)
	}

	return capLoc.SetService(name)
}

func (loc Location) SetID(id peer.ID) error {
	capLoc, err := loc.Location()
	if err != nil {
		return fmt.Errorf("fail to set peer ID: %w", err)
	}

	return capLoc.SetId(id.String())
}

func (loc Location) SetMaddrs(maddrs []ma.Multiaddr) error {
	capLoc, err := loc.Location()
	if err != nil {
		return fmt.Errorf("fail to create location: %w", err)
	}

	capMaddrs, err := capLoc.NewMaddrs(int32(len(maddrs)))
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
	capLoc, err := loc.Location()
	if err != nil {
		return fmt.Errorf("fail to create location: %w", err)
	}

	if err := capLoc.SetAnchor(anchor); err != nil {
		return fmt.Errorf("fail to set anchor in capnp Location: %w", err)
	}
	return nil
}

func (loc Location) SetCustom(custom capnp.Ptr) error {
	capLoc, err := loc.Location()
	if err != nil {
		return fmt.Errorf("fail to create location: %w", err)
	}

	if err := capLoc.SetCustom(custom); err != nil {
		return fmt.Errorf("fail to set custom in capnp Location: %w", err)
	}
	return nil
}

func (loc Location) Maddrs() ([]ma.Multiaddr, error) {
	capLoc, err := loc.Location()
	if err != nil {
		return nil, fmt.Errorf("failed to read location: %w", err)
	}

	capMaddrs, err := capLoc.Maddrs()
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
