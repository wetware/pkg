package service

import (
	"context"
	"fmt"

	"capnproto.org/go/capnp/v3"
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

	topicName, err := topic.Name(ctx)
	if err != nil {
		fut := capnp.ErrorAnswer(capnp.Method{}, fmt.Errorf("failed to read topic name: %w", err)).Future()
		iterator := casm.Iterator[Location]{
			Future: casm.Future{Future: fut},
		}
		cancel()
		return iterator, func() {}
	}

	handler := handler{ch: make(chan Location, 32), topic: topicName}

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

type handler struct {
	ch    chan Location
	topic string
}

func (h handler) Shutdown() { close(h.ch) }

func (h handler) Next() (b Location, ok bool) {
	b, ok = <-h.ch
	return
}

func (h handler) Send(ctx context.Context, call chan_api.Sender_send) error {
	// copy send arguments - TODO: use capnp message reference api
	ptr, err := call.Args().Value()
	if err != nil {
		return fmt.Errorf("failed to extract value: %w", err)
	}

	_, seg := capnp.NewSingleSegmentMessage(nil)
	capSloc, err := api.NewSignedLocation(seg)
	if err != nil {
		return fmt.Errorf("failed to create a signed location: %w", err)
	}

	if err := capSloc.ToPtr().Struct().CopyFrom(ptr.Struct()); err != nil {
		return fmt.Errorf("failed to copy/marshal signed location: %w", err)
	}

	// validate
	sloc := Location{SignedLocation: capSloc}
	if err := sloc.Validate(h.topic); err != nil {
		return fmt.Errorf("failed to validate location: %w", err)
	}

	select {
	case h.ch <- Location{SignedLocation: capSloc}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
