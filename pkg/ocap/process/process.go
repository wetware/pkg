package process

import (
	"context"

	"capnproto.org/go/capnp/v3/server"
	api "github.com/wetware/ww/internal/api/proc"
)

// TODO(someday):  JITCompiler executor that accepts Go source code
//                 or ASM, and executes it in a Func.

// Func specifies a process that runs in a native goroutine.
type Func func() error

// P is the basic process capability, from which all others are
// derived.  Processes are asynchronous and concurrent, and can
// be waited upon to complete.
type P api.P

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
//
// It is f's responsibility to terminate promptly when ctx expires.
func New(f Func) P {
	return NewWithPolicy(f, nil)
}

func NewWithPolicy(f Func, p *server.Policy) P {
	done := make(chan struct{})
	proc := process{done: done}

	go func() {
		defer close(done)
		proc.err = f()
	}()

	return P(api.P_ServerToClient(&proc, p))
}

// Wait blocks until the process terminates or the context expires,
// whichever comes first.  Wait returns any error returned by the
// process, or by the context's Err() method, if the context expired.
// Context errors are returned as expected.
//
// Wait is safe to call from multiple goroutines.
func (p P) Wait(ctx context.Context) error {
	f, release := api.P(p).Wait(ctx, nil)
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

type process struct {
	done <-chan struct{}
	err  error
}

func (p *process) Wait(ctx context.Context, call api.P_wait) error {
	select {
	case <-p.done:
		return p.err

	case <-ctx.Done():
		return ctx.Err()
	}
}
