package pubsub

import (
	"context"

	capnp "capnproto.org/go/capnp/v3"

	"github.com/wetware/ww/internal/api/pubsub"
)

type PubSub pubsub.PubSub

func (ps PubSub) Join(ctx context.Context, topic string) (FutureTopic, capnp.ReleaseFunc) {
	f, release := (pubsub.PubSub)(ps).Join(ctx, func(ps pubsub.PubSub_join_Params) error {
		return ps.SetName(topic)
	})

	return FutureTopic(f), release
}

type FutureTopic pubsub.PubSub_join_Results_Future

func (ft FutureTopic) Topic() Topic {
	return Topic((pubsub.PubSub_join_Results_Future)(ft).Topic())
}

func (ft FutureTopic) Struct() (Topic, error) {
	res, err := (pubsub.PubSub_join_Results_Future)(ft).Struct()
	if err != nil {
		return Topic{}, err
	}

	return Topic(res.Topic()), nil
}

type Topic pubsub.Topic

func (t Topic) Publish(ctx context.Context, b []byte) error {
	f, release := (pubsub.Topic)(t).Publish(ctx, func(ps pubsub.Topic_publish_Params) error {
		return ps.SetMsg(b)
	})
	defer release()

	_, err := f.Struct()
	return err
}

func (t Topic) Subscribe() Subscription {
	return newSubscription((pubsub.Topic)(t))
}

func (t Topic) Close() error { // TODO:  rename to Cancel and remove error?
	t.Client.Release()
	return nil
}
