package pubsub

import (
	"context"

	"capnproto.org/go/capnp/v3"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	api "github.com/wetware/ww/internal/api/pubsub"
	"golang.org/x/sync/semaphore"
)

type handler struct {
	cq chan struct{}
	ms chan []byte
}

func newHandler() handler {
	return handler{
		cq: make(chan struct{}),
		ms: make(chan []byte),
	}
}

func (h handler) Shutdown() {
	select {
	case <-h.cq:
		return
	default:
		close(h.cq)
	}
}

func (h handler) Handle(ctx context.Context, call api.Topic_Handler_handle) error {
	b, err := call.Args().Msg()
	if err != nil {
		return err
	}

	select {
	case h.ms <- b:
		return nil
	case <-h.cq:
		return ErrClosed
	case <-ctx.Done():
		return ctx.Err()
	}
}

type Subscription struct {
	h handler

	err     error // future resolution error
	f       *capnp.Future
	release capnp.ReleaseFunc
}

func newSubscription(t api.Topic) *Subscription {
	var (
		h          = newHandler()
		c          = api.Topic_Handler_ServerToClient(h, &defaultPolicy)
		f, release = t.Subscribe(
			context.Background(),
			func(ps api.Topic_subscribe_Params) error {
				return ps.SetHandler(c)
			})
	)

	return &Subscription{
		h:       h,
		f:       f.Future,
		release: release,
	}
}

func (s *Subscription) Cancel() {
	if s.release != nil {
		s.release()
	}

	s.h.Shutdown()
}

func (s *Subscription) Next(ctx context.Context) ([]byte, error) {
	if err := s.Resolve(ctx); err != nil {
		return nil, err
	}

	select {
	case b := <-s.h.ms:
		return b, nil

	case <-s.h.cq:
		return nil, ErrClosed

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Resolve blocks until the subscription is ready, the underlying
// RPC call fails, or the context expires. If the RPC call fails,
// the subscription is automatically canceled.
func (s *Subscription) Resolve(ctx context.Context) error {
	if s.release != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-s.f.Done():
			_, s.err = s.f.Struct()
			s.release()

			// free memory
			s.release = nil
			s.f = nil
		}
	}

	return s.err
}

type subHandler api.Topic_Handler

func (sh subHandler) Handle(ctx context.Context, sub *pubsub.Subscription) {
	ctx, cancel := context.WithCancel(ctx)
	defer sh.Client.Release()
	defer sub.Cancel()
	defer cancel()

	var (
		weight = int64(defaultPolicy.MaxConcurrentCalls)
		sem    = semaphore.NewWeighted(weight)
	)

	for {
		m, err := sub.Next(ctx)
		if err != nil {
			return
		}

		if err = sem.Acquire(ctx, 1); err != nil {
			return
		}

		go sh.send(ctx, m,
			func() { sem.Release(1) },
			cancel)
	}
}

func (sh subHandler) send(ctx context.Context, m *pubsub.Message, done, abort func()) {
	defer done()

	h := api.Topic_Handler(sh)
	f, release := h.Handle(ctx, func(ps api.Topic_Handler_handle_Params) error {
		return ps.SetMsg(m.Data)
	})
	defer release()

	// Abort the subscription if we receive a 'call on released client' exception.
	// This signals that the remote end has canceled their subscription.
	//
	// TODO:  test specifically for 'capnp: call on released client'.
	if _, err := f.Struct(); err != nil {
		abort()
	}
}
