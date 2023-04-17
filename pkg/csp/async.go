package csp

import (
	"container/list"
	"context"
	"errors"
	"sync"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/ww/internal/api/channel"
)

var _ ChanServer = (*AsyncChan)(nil)

// AsyncChan is a synchronous channel server. Both senders
// and receivers will block until a matching call arrives.
//
// The zero-value AsyncChan is ready to use.
type AsyncChan struct {
	mu                 sync.Mutex
	senders, receivers list.List
	signal, done       chan struct{}
	buffer             chan capnp.Ptr
	bufferLen, size    int
}

func (ch *AsyncChan) Close(ctx context.Context, call MethodClose) error {

	ch.initChannels()

	select {
	case <-ch.done:
		return errors.New("already closed")

	default:
		close(ch.done)
		return nil
	}
}

func (ch *AsyncChan) initChannels() {
	if ch.done == nil {
		ch.done = make(chan struct{})
		ch.buffer = make(chan capnp.Ptr, ch.bufferLen)
	}
}

func (ch *AsyncChan) Send(ctx context.Context, call MethodSend) error {

	ch.initChannels()

	ch.mu.Lock()
	defer ch.mu.Unlock()

	// Fast path:
	if ch.size < ch.bufferLen && ch.senders.Len() == 0 {
		val, err := call.Args().Value()
		if err != nil {
			return err
		}

		// Never will hang
		ch.buffer <- val
		ch.size += 1

		// TODO: signal any other waiting senders/recievers

		return err
	}

	// Slow path

	// Add the value to the send-queue, and wait for it to be picked up by
	// a receiver.
	// pending, err := ch.pushSend(ctx, call)
	// if err != nil {
	// 	return err
	// }

	// call.Go()

	// select {
	// case <-ch.done:
	// 	ch.mu.Lock()
	// 	defer ch.mu.Unlock()

	// 	ch.senders.Remove(pending.Sender)
	// 	return errors.New("closed")

	// case <-pending.Done:
	// 	// We're done; value was received.

	// case <-ctx.Done():
	// 	// Lock MUST be acquired before checking recved. The call to Recv()
	// 	// may still be performing work, and may need to return an error.
	// 	ch.mu.Lock()
	// 	defer ch.mu.Unlock()

	// 	select {
	// 	case <-ch.done:
	// 		ch.senders.Remove(pending.Sender)
	// 		return errors.New("closed")
	// 	default:
	// 	}

	// 	select {
	// 	case <-pending.Done:
	// 		// Received after we were canceled.  Rather than trying to fix
	// 		// up the queue, just pretend we didn't notice the cancelation.

	// 	default:
	// 		err = ctx.Err()
	// 		ch.senders.Remove(pending.Sender)
	// 	}
	// }

	return err
}

// Push the sender onto the queue and signal any receivers that a sender
// is ready.
//
// Callers MUST NOT hold mu.
func (ch *AsyncChan) pushSend(ctx context.Context, call MethodSend) (pendingSend, error) {
	// Do this first.  If something goes wrong, we can still back out without
	// affecting the queue's state.
	val, err := call.Args().Value()
	if err != nil {
		return pendingSend{}, err
	}

	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.initChannels()

	// TODO(performance):  profile & determine whether to use sync.Pool
	recved := make(chan struct{})
	elem := ch.senders.PushBack(sender{
		val:    val,
		recved: recved,
	})

	// signal that a sender is ready.
	select {
	case ch.signal <- struct{}{}:
	default: // TODO: Why do we need a select statement here?
	}

	return pendingSend{Sender: elem, Done: recved}, nil
}

func (ch *AsyncChan) Recv(ctx context.Context, call MethodRecv) error {
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
			return err
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
func (ch *AsyncChan) wait(ctx context.Context) error {
	ch.initChannels()

	ch.mu.Unlock()
	defer ch.mu.Lock()

	select {
	case <-ch.done:
		return errors.New("closed")
	case <-ch.signal:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// type pendingSend struct {
// 	Sender *list.Element
// 	Done   <-chan struct{}
// }

// type sender struct {
// 	val    capnp.Ptr
// 	recved chan<- struct{}
// }

// func (s sender) Bind(res api.Recver_recv_Results) (err error) {
// 	if err = res.SetValue(s.val); err == nil {
// 		close(s.recved)
// 	}

// 	return
// }

func (ch *AsyncChan) NewSender(ctx context.Context, call MethodNewSender) error {
	res, err := call.AllocResults()
	if err == nil {
		err = res.SetSender(api.Sender_ServerToClient(ch))
	}
	return err
}

func (ch *AsyncChan) NewRecver(ctx context.Context, call MethodNewRecver) error {
	res, err := call.AllocResults()
	if err == nil {
		err = res.SetRecver(api.Recver_ServerToClient(ch))
	}
	return err
}

func (ch *AsyncChan) NewCloser(ctx context.Context, call MethodNewCloser) error {
	res, err := call.AllocResults()
	if err == nil {
		err = res.SetCloser(api.Closer_ServerToClient(ch))
	}
	return err
}

func (ch *AsyncChan) NewSendCloser(ctx context.Context, call MethodNewSendCloser) error {
	res, err := call.AllocResults()
	if err == nil {
		err = res.SetSendCloser(api.SendCloser_ServerToClient(ch))
	}
	return err
}
