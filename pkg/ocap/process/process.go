package process

import (
	"context"

	"github.com/wetware/ww/internal/api/proc"
)

// P is the basic process capability, from which all others are
// derived.  Processes are asynchronous and concurrent, and can
// be waited upon to complete.
type P proc.P

func (p P) AddRef() P {
	return P{
		Client: p.Client.AddRef(),
	}
}

func (p P) Release() {
	p.Client.Release()
}

// New calls 'f' in a separate goroutine, and returns a process
// whose Wait() method returns the error returned by 'f'.
func New(f func() error) P {
	var (
		done = make(chan struct{})
		b    = barrier{done: done}
	)

	go func() {
		defer close(done)
		b.err = f()
	}()

	return P(proc.P_ServerToClient(&b, nil))
}

// Wait blocks until the process terminates or the context expires,
// whichever comes first.  Wait returns any error returned by the
// process, or by the context's Err() method, if the context expired.
// Context errors are returned as expected.
//
// Wait is safe to call from multiple goroutines.
func (p P) Wait(ctx context.Context) error {
	f, release := proc.P(p).Wait(ctx, nil)
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

type barrier struct {
	done <-chan struct{}
	err  error
}

func (b *barrier) Wait(ctx context.Context, call proc.P_wait) error {
	call.Ack()

	select {
	case <-b.done:
		return b.err

	case <-ctx.Done():
		return ctx.Err()
	}
}
