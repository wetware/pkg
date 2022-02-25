package pubsub

import (
	"context"

	capnp "capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"

	api "github.com/wetware/ww/internal/api/pubsub"
)

type PubSub api.PubSub

func (ps PubSub) Join(ctx context.Context, topic string) (FutureTopic, capnp.ReleaseFunc) {
	f, release := (api.PubSub)(ps).Join(ctx, func(ps api.PubSub_join_Params) error {
		return ps.SetName(topic)
	})

	return FutureTopic(f), release
}

func (ps PubSub) AddRef() PubSub {
	return PubSub(api.PubSub(ps).AddRef())
}

func (ps PubSub) Release() { ps.Client.Release() }

type FutureTopic api.PubSub_join_Results_Future

func (ft FutureTopic) Topic() Topic {
	return Topic(api.PubSub_join_Results_Future(ft).Topic())
}

func (ft FutureTopic) Struct() (Topic, error) {
	res, err := (api.PubSub_join_Results_Future)(ft).Struct()
	if err != nil {
		return Topic{}, err
	}

	return Topic(res.Topic()), nil
}

type Topic api.Topic

func (t Topic) Publish(ctx context.Context, b []byte) error {
	f, release := (api.Topic)(t).Publish(ctx, func(ps api.Topic_publish_Params) error {
		return ps.SetMsg(b)
	})
	defer release()

	_, err := f.Struct()
	return err
}

func (t Topic) Subscribe(ctx context.Context, ch chan<- []byte) (cancel func(), err error) {
	hc := api.Topic_Handler_ServerToClient(handler{
		ms:      ch,
		release: t.AddRef().Release,
	}, &server.Policy{
		MaxConcurrentCalls: cap(ch),
		AnswerQueueSize:    cap(ch),
	})
	defer hc.Release() // ensure topic ref is released if we return an error

	f, release := api.Topic(t).Subscribe(ctx, func(ps api.Topic_subscribe_Params) error {
		ps.SetBufSize(uint8(cap(ch)))
		return ps.SetHandler(hc.AddRef())
	})
	defer release()

	select {
	case <-f.Done():
		_, err = f.Struct()
	case <-ctx.Done():
		err = ctx.Err()
	}

	return hc.AddRef().Release, err
}

func (t Topic) Release() { t.Client.Release() }

func (t Topic) AddRef() Topic {
	return Topic(api.Topic(t).AddRef())
}
