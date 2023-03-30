package service

import (
	"context"
	"fmt"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	casm "github.com/wetware/casm/pkg"
	chan_api "github.com/wetware/ww/internal/api/channel"
	ps_api "github.com/wetware/ww/internal/api/pubsub"
	api "github.com/wetware/ww/internal/api/service"
	"github.com/wetware/ww/pkg/pubsub"
)

type Registry api.Registry

func (c Registry) Release() {
	api.Registry(c).Release()
}

func (c Registry) Provide(ctx context.Context, topic pubsub.Topic, loc Location) (casm.Future, capnp.ReleaseFunc) {
	ctx, cancel := context.WithCancel(ctx)

	fut, release := api.Registry(c).Provide(ctx, func(ps api.Registry_provide_Params) error {
		if err := ps.SetTopic(ps_api.Topic(topic)); err != nil {
			return err
		}
		return ps.SetLocation(loc.SignedLocation)
	})

	return casm.Future(fut), func() {
		cancel()
		release()
	}
}

func (c Registry) FindProviders(ctx context.Context, topic pubsub.Topic) (casm.Iterator[Location], capnp.ReleaseFunc) {
	ctx, cancel := context.WithCancel(ctx)

	handler := make(handler, 32)

	fut, release := api.Registry(c).FindProviders(ctx, func(ps api.Registry_findProviders_Params) error {
		if err := ps.SetTopic(ps_api.Topic(topic)); err != nil {
			return err
		}
		return ps.SetChan(chan_api.Sender_ServerToClient(handler))
	})

	iterator := casm.Iterator[Location]{
		Future: casm.Future(fut),
		Seq:    handler, // TODO: decide buffer size
	}

	return iterator, func() {
		cancel()
		release()
	}
}

type handler chan Location

func (ch handler) Shutdown() { close(ch) }

func (ch handler) Next() (b Location, ok bool) {
	b, ok = <-ch
	return
}

func (ch handler) Send(ctx context.Context, call chan_api.Sender_send) error {
	// copy send arguments - TODO: use capnp message reference api
	b, err := call.Args().Message().Marshal()
	msg, err := capnp.Unmarshal(b)
	if err != nil {
		return fmt.Errorf("failed to copy message: %w", err)
	}
	args, err := chan_api.ReadRootSender_send_Params(msg)
	if err != nil {
		return fmt.Errorf("failed to read copied message: %w", err)
	}

	// extract location and send to user channel
	ptr, err := args.Value()
	if err == nil {
		select {
		case ch <- Location{SignedLocation: api.SignedLocation(ptr.Struct())}:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return err
}

type Location struct {
	api.SignedLocation
}

func NewLocation() (Location, error) {
	_, seg := capnp.NewSingleSegmentMessage(nil)
	loc, err := api.NewSignedLocation(seg)
	if err != nil {
		return Location{}, fmt.Errorf("failed to create location: %w", err)
	}
	return Location{SignedLocation: loc}, nil
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

func (loc Location) VerifySinature() (bool, error) {
	capLoc, err := loc.Location()
	if err != nil {
		return false, fmt.Errorf("failed to read location: %w", err)
	}

	b, err := capLoc.Message().Marshal()
	if err != nil {
		return false, fmt.Errorf("failed to marshal location: %w", err)
	}

	idBytes, err := capLoc.Id()
	if err != nil {
		return false, fmt.Errorf("failed to extract peer ID: %w", err)
	}
	peerID := peer.ID(idBytes)
	pubKey, err := peerID.ExtractPublicKey()
	if err != nil {
		return false, fmt.Errorf("failed to extract public key: %w", err)
	}

	sig, err := loc.Signature()
	if err != nil {
		return false, fmt.Errorf("failed to extract signature: %w", err)
	}

	return pubKey.Verify(b, sig)
}

func (loc Location) SetMaddrs(maddrs []ma.Multiaddr) error {
	capLoc, err := loc.NewLocation()
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
	capLoc, err := loc.NewLocation()
	if err != nil {
		return fmt.Errorf("fail to create location: %w", err)
	}

	if err := capLoc.SetAnchor(anchor); err != nil {
		return fmt.Errorf("fail to set anchor in capnp Location: %w", err)
	}
	return nil
}

func (loc Location) SetCustom(custom capnp.Ptr) error {
	capLoc, err := loc.NewLocation()
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
