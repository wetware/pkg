package pubsub

import (
	"context"
	"sync"

	capnp "capnproto.org/go/capnp/v3"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	api "github.com/wetware/ww/internal/api/pubsub"
	"golang.org/x/sync/semaphore"
)

const subBufSize = 32

type Subscription interface {
	Next(context.Context) ([]byte, error)
	Cancel()
}

type subscription struct {
	cq chan struct{}
	ms chan []byte

	once    sync.Once
	release capnp.ReleaseFunc
}

func (sub *subscription) Cancel() {
	sub.once.Do(func() {
		close(sub.cq)
		sub.release()
	})
}

func (sub *subscription) Next(ctx context.Context) (b []byte, err error) {
	select {
	case b = <-sub.ms:
	case <-sub.cq:
		err = ErrClosed
	case <-ctx.Done():
		err = ctx.Err()
	}

	return
}

func (sub *subscription) Handle(ctx context.Context, call api.Topic_Handler_handle) error {
	call.Ack()

	b, err := call.Args().Msg()
	if err != nil {
		return err
	}

	select {
	case <-sub.cq:
		return ErrClosed

	case sub.ms <- b:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
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
