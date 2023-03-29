package discovery

import (
	"context"
	"time"

	"capnproto.org/go/capnp/v3"
	"github.com/wetware/ww/internal/api/channel"
	api "github.com/wetware/ww/internal/api/discovery"
	"github.com/wetware/ww/pkg/pubsub"
)

type DiscoveryServiceServer struct {
	pubsub.Router
}

func (s *DiscoveryServiceServer) Client() capnp.Client {
	return capnp.Client(api.DiscoveryService_ServerToClient(s))
}

func (s *DiscoveryServiceServer) Discovery() DiscoveryService {
	return DiscoveryService(s.Client())
}

func (s *DiscoveryServiceServer) Provider(_ context.Context, call api.DiscoveryService_provider) error {
	name, err := call.Args().Name()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	topic, release := s.Join(ctx, name)

	provider := ProviderServer{
		name:  name,
		Topic: topic,
		release: func() {
			cancel()
			release()
		},
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetProvider(api.Provider_ServerToClient(&provider))
}

func (s *DiscoveryServiceServer) Locator(_ context.Context, call api.DiscoveryService_locator) error {
	name, err := call.Args().Name()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	topic, release := s.Join(ctx, name)
	provider := LocatorServer{
		name:  name,
		Topic: &topic,
		release: func() {
			cancel()
			release()
		},
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

	call.Go()
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
	*pubsub.Topic
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

	time.Sleep(time.Second)

	// publish a request
	call.Go()
	if err := s.Topic.Publish(ctx, request); err != nil {
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
			response, err := msg.Response()
			if err != nil {
				return err
			}

			loc, err := response.Location()
			if err != nil {
				return err
			}

			fut, release := sender.Send(ctx, func(ps channel.Sender_send_Params) error {
				return ps.SetValue(loc.ToPtr())
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

func encodeRequest(call api.Locator_findProviders) ([]byte, error) {
	_, seg := capnp.NewSingleSegmentMessage(nil)
	msg, err := api.NewRootMessage(seg)
	if err != nil {
		return nil, err
	}

	request, err := msg.NewRequest()
	if err != nil {
		return nil, err
	}

	return request.Message().MarshalPacked()
}

func encodeResponse(call api.Provider_provide) ([]byte, error) {
	loc, err := call.Args().Location()
	if err != nil {
		return nil, err
	}

	_, seg := capnp.NewSingleSegmentMessage(nil)
	msg, err := api.NewRootMessage(seg)
	if err != nil {
		return nil, err
	}

	response, err := msg.NewResponse()
	if err != nil {
		return nil, err
	}

	if err := response.SetLocation(loc); err != nil {
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
