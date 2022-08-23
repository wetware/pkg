package process

import (
	"context"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/ww/internal/api/proc"
)

// Executor spawns native Go processes with the supplied server
// policy.  Note that unlike most executors, it is not itself a
// wire-compatible capability.  This is because there is no way
// to serialize a native Go function object.
type Executor struct{}

// Exec calls f(ctx) in a separate goroutine, and returns a process
// whose Wait() method returns f's error (possibly wrapped in a capnp
// exception type).
//
// f MUST return promptly when ctx expires.
func (exec Executor) Exec(ctx context.Context, f func(context.Context) error) Proc {
	handle := Executor{}.Go(context.Background(), f)
	return Proc(api.Waiter_ServerToClient(handle))
}

// Go runs f in a separate goroutine and returns its Handle.
func (exec Executor) Go(ctx context.Context, f func(context.Context) error) *Handle {
	ctx, cancel := context.WithCancel(ctx)

	done := make(chan struct{})
	handle := &Handle{
		cancel:  cancel,
		closing: ctx.Done(),
		closed:  done,
	}

	go func() {
		defer close(done)
		handle.err = f(ctx)
	}()

	return handle
}

// Proc is the basic process capability, from which all others are
// derived.  Processes are asynchronous and concurrent, and can be
// waited upon to complete.
type Proc api.Waiter

func (p Proc) AddRef() Proc {
	return Proc(capnp.Client(p).AddRef())
}

func (p Proc) Release() {
	capnp.Client(p).Release()
}

// New is a convenience method for a process with the root context
// and the default policy.  It is equivalent to:
//
//    Executor{}.Spawn(context.Background(), f)
func New(f func(context.Context) error) Proc {
	return Executor{}.Exec(context.Background(), f)
}

// Wait blocks until the process terminates or the context expires,
// whichever comes first.  Wait returns any error returned by the
// process, or by the context's Err() method, if the context expired.
// Context errors are returned as expected.
//
// Wait is safe to call from multiple goroutines.
func (p Proc) Wait(ctx context.Context) error {
	f, release := api.Waiter(p).Wait(ctx, nil)
	defer release()

	select {
	case <-f.Done():
	case <-ctx.Done():
	}

	// The future may have resolved due to a canceled context, in which
	// case there is a race-condition in the above select.
	if ctx.Err() != nil {
		return ctx.Err()
	}

	_, err := f.Struct()
	return err
}

// Handle is a reference to a running process.  It implements the
// Waiter capability interface.
//
// To avoid resource leaks, the underlying process' releasing all
// references to the Handle causes its context to be canceled, in
// turn signalling to the process that it should return.
type Handle struct {
	cancel          context.CancelFunc
	closing, closed <-chan struct{}
	err             error
}

// Shutdown sends a signal to the goroutine to terminate ongoing work
// and return.
//
// Ordering:  <-h.Closing() happens before h.Shutdown() returns.
func (h *Handle) Shutdown() {
	h.cancel()
}

// Closing returns a channel that is closed after the process has
// received the shutdown signal.
//
// Ordering:  <-h.Closing() happens before <-h.Done().
func (h *Handle) Closing() <-chan struct{} {
	return h.closing
}

// Done returns a channel that is closed after the process terminates.
func (h *Handle) Done() <-chan struct{} {
	return h.closed
}

// Err returns the process error.  The error is guaranteed to be nil
// if p.Err() returns before <-h.Done().
func (p *Handle) Err() error {
	select {
	case <-p.closed:
		return p.err
	default:
		return nil
	}
}

// Wait is the RPC handler for the process.Waiter.wait() method.
func (h *Handle) Wait(ctx context.Context, call api.Waiter_wait) error {
	select {
	case <-h.closed:
		return h.err

	case <-ctx.Done():
		return ctx.Err()
	}
}
