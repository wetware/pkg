package pubsub

import (
	"context"

	capnp "capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	"golang.org/x/sync/semaphore"

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

func (t Topic) Name(ctx context.Context) (string, error) {
	f, release := (api.Topic)(t).Name(ctx, nil)
	defer release()

	res, err := f.Struct()
	if err != nil {
		return "", err
	}

	return res.Name()
}

func (t Topic) Publish(ctx context.Context, b []byte) error {
	f, release := (api.Topic)(t).Publish(ctx, func(ps api.Topic_publish_Params) error {
		return ps.SetMsg(b)
	})
	defer release()

	_, err := f.Struct()
	return err
}

func (t Topic) Subscribe(ctx context.Context, ch chan<- []byte) (release capnp.ReleaseFunc, err error) {
	h := api.Topic_Handler_ServerToClient(handler{
		ms:      ch,
		release: t.AddRef().Release,
	}, &server.Policy{
		MaxConcurrentCalls: cap(ch),
	})

	// Limit the number of in-flight requests to half of the channel's capacity.
	// This prevents the server from running out of memory due to the unbounded
	// send-queue in capnp.
	//
	// TODO:  make this configurable.
	h.Client.SetFlowLimiter(newFlowLimiter(cap(ch) / 2))

	f, release := api.Topic(t).Subscribe(ctx, func(ps api.Topic_subscribe_Params) error {
		return ps.SetHandler(h)
	})
	defer release()

	if _, err = f.Struct(); err == nil {
		release = h.Release
	}

	return
}

func (t Topic) Release() { t.Client.Release() }

func (t Topic) AddRef() Topic {
	return Topic(api.Topic(t).AddRef())
}

type handler struct {
	ms      chan<- []byte
	release capnp.ReleaseFunc
}

func (h handler) Shutdown() {
	close(h.ms)
	h.release()
}

func (h handler) Handle(_ context.Context, call api.Topic_Handler_handle) error {
	b, err := call.Args().Msg()
	if err != nil {
		return err
	}

	select {
	case h.ms <- b:
	default:
	}

	return nil
}

type flowLimiter semaphore.Weighted

func newFlowLimiter(limit int) *flowLimiter {
	sem := semaphore.NewWeighted(int64(max(1, limit)))
	return (*flowLimiter)(sem)
}

func (f *flowLimiter) StartMessage(ctx context.Context, size uint64) (gotResponse func(), err error) {
	if err = (*semaphore.Weighted)(f).Acquire(ctx, 1); err == nil {
		gotResponse = func() { (*semaphore.Weighted)(f).Release(1) }
	}

	return
}

func max(x, y int) int {
	if x < y {
		return y
	}

	return x
}
