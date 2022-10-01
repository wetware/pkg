package discovery

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"github.com/wetware/ww/internal/api/channel"
	api "github.com/wetware/ww/internal/api/service"
	"github.com/wetware/ww/pkg/pubsub"
)

type DiscoveryServiceServer struct {
	pubsub.Joiner
}

func (s *DiscoveryServiceServer) Client() capnp.Client {
	return capnp.Client(api.DiscoveryService_ServerToClient(s))
}

func (s *DiscoveryServiceServer) Provider(ctx context.Context, call api.DiscoveryService_provider) error {
	name, err := call.Args().Name()
	if err != nil {
		return err
	}

	topic, release := s.Join(ctx, name)
	provider := ProviderServer{
		name:    name,
		Topic:   topic,
		release: release,
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetProvider(api.Provider_ServerToClient(&provider))
}

func (s *DiscoveryServiceServer) Locator(ctx context.Context, call api.DiscoveryService_locator) error {
	name, err := call.Args().Name()
	if err != nil {
		return err
	}

	topic, release := s.Join(ctx, name)
	provider := LocatorServer{
		name:    name,
		Topic:   topic,
		release: release,
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetLocator(api.Locator_ServerToClient(&provider))
}

type ProviderServer struct {
	pubsub.Topic
	name    string
	release capnp.ReleaseFunc
}

func (s *ProviderServer) Shutdown() {
	s.release()
}

func (s *ProviderServer) Provide(ctx context.Context, call api.Provider_provide) error {
	response, err := encodeResponse(call)
	if err != nil {
		return err
	}
	// subscribe to topic
	sub, release := s.Topic.Subscribe(ctx)
	defer release()

	for b := sub.Next(); b != nil; b = sub.Next() {
		msg, err := decodeMessage(b)
		if err != nil {
			return err
		}

		if msg.Which() == api.Message_Which_request {
			if err := s.Topic.Publish(ctx, response); err != nil {
				return err
			}
		}
	}

	return sub.Err()
}

type LocatorServer struct {
	pubsub.Topic
	name    string
	release capnp.ReleaseFunc
}

func (s *LocatorServer) Shutdown() {
	s.release()
}

func (s *LocatorServer) FindProviders(ctx context.Context, call api.Locator_findProviders) error {
	request, err := encodeRequest(call)
	if err != nil {
		return err
	}
	sub, release := s.Topic.Subscribe(ctx)
	defer release()

	// publish a request
	if err := s.Topic.Publish(ctx, request); err != nil {
		return err
	}
	// wait for responses or until context is canceled
	sender := call.Args().Chan()

	for msg := sub.Next(); msg != nil; msg = sub.Next() {
		capMsg, err := decodeMessage(msg)
		if err != nil {
			return err
		}
		if capMsg.Which() == api.Message_Which_response {
			response, err := capMsg.Response()
			if err != nil {
				return err
			}
			addrs, err := response.Addrs()
			if err != nil {
				return err
			}

			for i := 0; i < addrs.Len(); i++ {
				fut, release := sender.Send(ctx, func(ps channel.Sender_send_Params) error {
					return ps.SetValue(addrs.At(i).ToPtr())
				})
				defer release()

				_, err := fut.Struct()
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func encodeRequest(call api.Locator_findProviders) ([]byte, error) {
	_, seg := capnp.NewSingleSegmentMessage(nil)
	request, err := api.NewMessage_Request(seg)
	if err != nil {
		return nil, err
	}

	return request.Message().MarshalPacked()
}

func encodeResponse(call api.Provider_provide) ([]byte, error) {
	addrs, err := call.Args().Addrs()
	if err != nil {
		return nil, err
	}

	_, seg := capnp.NewSingleSegmentMessage(nil)
	response, err := api.NewMessage_Response(seg)
	if err != nil {
		return nil, err
	}

	if err := response.SetAddrs(addrs); err != nil {
		return nil, err
	}

	return response.Message().MarshalPacked()

}

func decodeMessage(b []byte) (api.Message, error) {
	msg, err := capnp.UnmarshalPacked(b)
	if err != nil {
		return api.Message{}, err
	}

	return api.ReadRootMessage(msg)
}
