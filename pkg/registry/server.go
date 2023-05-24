package service

import (
	"context"
	"fmt"

	"capnproto.org/go/capnp/v3"
	"github.com/wetware/ww/internal/api/channel"
	api "github.com/wetware/ww/internal/api/registry"
	"github.com/wetware/ww/pkg/pubsub"
)

type Server struct{}

func (s Server) Registry() Registry {
	sds := RegistryServer{}
	return Registry(sds.Client())
}

type RegistryServer struct{}

func (s *RegistryServer) Client() capnp.Client {
	return capnp.Client(api.Registry_ServerToClient(s))
}

func (s *RegistryServer) Provide(ctx context.Context, call api.Registry_provide) error {
	e, err := call.Args().Envelope()
	if err != nil {
		return fmt.Errorf("failed to read location: %w", err)
	}

	response, err := encodeResponse(e)
	if err != nil {
		return err
	}

	topic := pubsub.Topic(call.Args().Topic())

	// subscribe to topic
	sub, release := topic.Subscribe(ctx)
	defer release()

	call.Go()
	for b := sub.Next(); b != nil; b = sub.Next() {
		msg, err := decodeMessage(b)
		if err != nil {
			return err
		}

		if msg.Which() == api.Message_Which_request {
			if err := topic.Publish(ctx, response); err != nil {
				return err
			}
		}
	}

	return sub.Err()
}

func (s *RegistryServer) FindProviders(ctx context.Context, call api.Registry_findProviders) error {
	request, err := encodeRequest(call)
	if err != nil {
		return err
	}

	topic := pubsub.Topic(call.Args().Topic())

	sub, release := topic.Subscribe(ctx)
	defer release()

	// publish a request
	call.Go()
	if err := topic.Publish(ctx, request); err != nil {
		return err
	}

	// wait for responses or until context is canceled
	sender := call.Args().Chan()

	for b := sub.Next(); b != nil; b = sub.Next() {
		msg, err := decodeMessage(b)
		if err != nil {
			return err
		}
		if msg.Which() == api.Message_Which_response {
			loc, err := msg.Response()
			if err != nil {
				return err
			}

			fut, release := sender.Send(ctx, func(ps channel.Sender_send_Params) error {
				_, seg := capnp.NewSingleSegmentMessage(nil)
				data, err := capnp.NewData(seg, loc)
				if err != nil {
					return err
				}

				return ps.SetValue(data.ToPtr())
			})
			defer release()

			_, err = fut.Struct()
			if err != nil {
				return err
			}

		}
	}
	return nil
}

func encodeRequest(call api.Registry_findProviders) ([]byte, error) {
	_, seg := capnp.NewSingleSegmentMessage(nil)
	msg, err := api.NewRootMessage(seg)
	if err != nil {
		return nil, err
	}

	return msg.Message().MarshalPacked()
}

func encodeResponse(e []byte) ([]byte, error) {
	_, seg := capnp.NewSingleSegmentMessage(nil)
	msg, err := api.NewRootMessage(seg)
	if err != nil {
		return nil, err
	}

	if err := msg.SetResponse(e); err != nil {
		return nil, err
	}

	return msg.Message().MarshalPacked()

}

func decodeMessage(b []byte) (api.Message, error) {
	msg, err := capnp.UnmarshalPacked(b)
	if err != nil {
		return api.Message{}, err
	}

	return api.ReadRootMessage(msg)
}
