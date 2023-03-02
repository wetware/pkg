package csp

import (
	"container/list"
	"context"
	"errors"
	"sync"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/ww/internal/api/channel"
)

var _ ChanServer = (*SyncServer)(nil)

// SyncServer is a synchronous channel server. Both senders
// and receivers will block until a matching call arrives.
//
// The zero-value SyncServer is ready to use.
type SyncServer struct {
	mu      sync.Mutex
	senders list.List
	signal  chan struct{}
}

func (*SyncServer) Cap() uint {
	return 0
}

func (ch *SyncServer) Close(ctx context.Context, call MethodClose) error {
	panic("not implemented yet")
}

func (ch *SyncServer) Send(ctx context.Context, call MethodSend) error {
	// Add the value to the send-queue, and wait for it to be picked up by
	// a receiver.
	pending, err := ch.pushSend(ctx, call)
	if err != nil {
		return err
	}

	call.Go()

	select {
	case <-pending.Done:
		// We're done; value was received.

	case <-ctx.Done():
		// Lock MUST be acquired before checking recved. The call to Recv()
		// may still be performing work, and may need to return an error.
		ch.mu.Lock()
		defer ch.mu.Unlock()

		select {
		case <-pending.Done:
			// Reveived after we were canceled.  Rather than trying to fix
			// up the queue, just pretend we didn't notice the cancelation.

		default:
			err = ctx.Err()
			ch.senders.Remove(pending.Sender)
		}
	}

	return err
}

// Push the sender onto the queue and signal any receivers that a sender
// is ready.
//
// Callers MUST NOT hold mu.
func (ch *SyncServer) pushSend(ctx context.Context, call MethodSend) (pendingSend, error) {
	if call.Args().Async() {
		return pendingSend{}, errors.New("async call to sync channel")
	}

	// Do this first.  If something goes wrong, we can still back out without
	// affecting the queue's state.
	val, err := call.Args().Value()
	if err != nil {
		return pendingSend{}, err
	}

	ch.mu.Lock()
	defer ch.mu.Unlock()

	if ch.signal == nil {
		ch.signal = make(chan struct{}, 1)
	}

	// TODO(performance):  profile & determine whether to use sync.Pool
	recved := make(chan struct{})
	elem := ch.senders.PushBack(sender{
		val:    val,
		recved: recved,
	})

	// signal that a sender is ready.
	select {
	case ch.signal <- struct{}{}:
	default:
	}

	return pendingSend{Sender: elem, Done: recved}, nil
}

func (ch *SyncServer) Recv(ctx context.Context, call MethodRecv) error {
	if call.Args().Async() {
		return errors.New("async call to sync channel")
	}

	// Do this first.  If something goes wrong, we can still back out
	// without affecting the queue's state.
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	ch.mu.Lock()
	defer ch.mu.Unlock()

	var next *list.Element
	for next = ch.senders.Front(); next == nil; next = ch.senders.Front() {
		// slow path; we're going to have to wait
		call.Go()

		// wait for sender; temporarily unlocks mu
		if err := ch.wait(ctx); err != nil {
			return err // always a context error
		}
	}

	// If we fail to bind the sender's value to the results struct,
	// then we do *not* want to dequeue the sender, so that another
	// receiver can try its luck.  In such cases, Recv's failure is
	// transparent to Send().
	if err = next.Value.(sender).Bind(res); err == nil {
		ch.senders.Remove(next) // commit
	}

	return err
}

// wait for a sender to signal that it has added itself to the queue.
//
// Callers MUST hold mu.
func (ch *SyncServer) wait(ctx context.Context) error {
	if ch.signal == nil {
		ch.signal = make(chan struct{}, 1)
	}

	ch.mu.Unlock()
	defer ch.mu.Lock()

	select {
	case <-ch.signal:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type pendingSend struct {
	Sender *list.Element
	Done   <-chan struct{}
}

type sender struct {
	val    capnp.Ptr
	recved chan<- struct{}
}

func (s sender) Bind(res api.Recver_recv_Results) (err error) {
	if err = res.SetValue(s.val); err == nil {
		close(s.recved)
	}

	return
}

func (ch *SyncServer) AsSender(ctx context.Context, call MethodAsSender) error {
	res, err := call.AllocResults()
	if err == nil {
		err = res.SetSender(api.Sender_ServerToClient(ch))
	}
	return err
}

func (ch *SyncServer) AsRecver(ctx context.Context, call MethodAsRecver) error {
	res, err := call.AllocResults()
	if err == nil {
		err = res.SetRecver(api.Recver_ServerToClient(ch))
	}
	return err
}

func (ch *SyncServer) AsCloser(ctx context.Context, call MethodAsCloser) error {
	res, err := call.AllocResults()
	if err == nil {
		err = res.SetCloser(api.Closer_ServerToClient(ch))
	}
	return err
}

func (ch *SyncServer) AsSendCloser(ctx context.Context, call MethodAsSendCloser) error {
	res, err := call.AllocResults()
	if err == nil {
		err = res.SetSendCloser(api.SendCloser_ServerToClient(ch))
	}
	return err
}