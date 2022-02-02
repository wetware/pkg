package pubsub

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	api "github.com/wetware/ww/internal/api/pubsub"
	"golang.org/x/sync/semaphore"
)

type handler struct {
	ms      chan<- []byte
	release capnp.ReleaseFunc
}

func (h handler) Shutdown() {
	close(h.ms)
	h.release()
}

func (h handler) Handle(ctx context.Context, call api.Topic_Handler_handle) error {
	b, err := call.Args().Msg()
	if err != nil {
		return err
	}

	select {
	case h.ms <- b:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

type Subscription struct {
	ms      <-chan []byte
	release capnp.ReleaseFunc
}

func newSubscription(ctx context.Context, t api.Topic, ms chan []byte) (*Subscription, error) {
	h := handler{
		ms:      ms,
		release: t.AddRef().Release,
	}

	hc := api.Topic_Handler_ServerToClient(h, &server.Policy{
		MaxConcurrentCalls: cap(ms),
		AnswerQueueSize:    cap(ms),
	})
	defer hc.Release() // ensure h.Shutdown is called on error; see return

	f, release := t.Subscribe(ctx,
		func(ps api.Topic_subscribe_Params) error {
			ps.SetBufSize(uint8(cap(ms)))
			return ps.SetHandler(hc.AddRef()) // NOTE: incr refcount
		})
	defer release()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-f.Done():
		if _, err := f.Struct(); err != nil {
			return nil, err
		}
	}

	return &Subscription{
		ms:      ms,
		release: hc.AddRef().Release, // offset the deferred hc.Release
	}, nil
}

func (s *Subscription) Cancel() { s.release() }

func (s *Subscription) Next(ctx context.Context) ([]byte, error) {
	select {
	case b, ok := <-s.ms:
		if ok {
			return b, nil
		}

		return nil, ErrClosed

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type subHandler struct {
	handler api.Topic_Handler
	buffer  *semaphore.Weighted
}

func (sh subHandler) Handle(ctx context.Context, sub *pubsub.Subscription) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		m, err := sub.Next(ctx)
		if err != nil {
			return
		}

		// TODO:  reduce goroutine count in order to conserve memory. Goroutines
		//        have an initial stack-size of 2kb, and we have O(n) goroutines
		//        per topic, where n is the number of messages.  As such, memory
		//        consumption will be highest at peak system load.
		//
		//        We should investigate reflect.Select as a means to achieve O(1)
		//        goroutines per topic.  During this investigation, we must check
		//        that reflect.Select does not introduce latent O(n) consumption,
		//        either.
		if err = sh.buffer.Acquire(ctx, 1); err == nil {
			go sh.send(ctx, m, cancel)
		}
	}
}

func (sh subHandler) send(ctx context.Context, m *pubsub.Message, abort func()) {
	defer sh.buffer.Release(1)

	f, release := sh.handler.Handle(ctx,
		func(ps api.Topic_Handler_handle_Params) error {
			return ps.SetMsg(m.Data)
		})
	defer release()

	select {
	case <-f.Done():
	case <-ctx.Done():
		return
	}

	// Abort the subscription if we receive a 'call on released client' exception.
	// This signals that the remote end has canceled their subscription.
	//
	// TODO:  test specifically for 'capnp: call on released client'.
	if _, err := f.Struct(); err != nil {
		abort()
	}
}
