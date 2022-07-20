package unix

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"syscall"
	"time"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/ww/internal/api/proc"
	"github.com/wetware/ww/pkg/vat"
	"github.com/wetware/ww/pkg/vat/cap/iostream"
	"github.com/wetware/ww/pkg/vat/cap/process"
)

type Proc api.Unix_Proc

func (p Proc) AddRef() Proc {
	return Proc{
		Client: p.Client.AddRef(),
	}
}

func (p Proc) Release() {
	p.Client.Release()
}

func (p Proc) Wait(ctx context.Context) error {
	return process.Proc(p).Wait(ctx)
}

func (p Proc) Signal(ctx context.Context, s syscall.Signal) (vat.Future, capnp.ReleaseFunc) {
	f, release := api.Unix_Proc(p).Signal(ctx, func(ps api.Unix_Proc_signal_Params) (err error) {
		switch s {
		case syscall.SIGINT:
			ps.SetSignal(api.Unix_Proc_Signal_sigINT)

		case syscall.SIGTERM:
			ps.SetSignal(api.Unix_Proc_Signal_sigTERM)

		case syscall.SIGKILL:
			ps.SetSignal(api.Unix_Proc_Signal_sigKILL)

		default:
			err = fmt.Errorf("unknown signal: %#x", int(s))
		}

		return
	})

	return vat.Future(f), release
}

// handle is a reference to a running the server implementation of UnixProc
type handle struct {
	Cmd *exec.Cmd
	*process.Handle
}

func (h *handle) Shutdown() {
	defer func() {
		h.Handle.Shutdown()
		<-h.Done() // wait for process to terminate
	}()

	// Attempt to gracefully shut down the process, but kill it if the
	// timeout is exceeded.
	if err := h.Cmd.Process.Signal(syscall.SIGTERM); err == nil {
		select {
		case <-h.Done():
		case <-time.After(time.Second * 5):
		}
	}
}

func (h *handle) bind(ctx context.Context, cmd api.Unix_Command) error {
	path, err := cmd.Path()
	if err != nil {
		return err
	}

	args, err := stringSlice(cmd.Args)
	if err != nil {
		return err
	}

	environment, err := stringSlice(cmd.Env)
	if err != nil {
		return err
	}

	/*
		The exec.Cmd instance must be created, and its fields populated, with
		a context that is expired by the Handdler.  Therefore, we perform the
		configuration of h.Cmd *inside* of the executor process.   To avoid a
		rance condition, we use a synchronization channel.
	*/
	var cherr = make(chan error, 1)

	h.Handle = process.Executor{}.Go(ctx, func(ctx context.Context) error {
		defer close(cherr)

		h.Cmd = exec.CommandContext(ctx, path, args...)
		h.Cmd.Env = environment
		h.Cmd.Stdin = input(ctx, iostream.Provider(cmd.Stdin()).AddRef())
		h.Cmd.Stdout = output(ctx, iostream.Stream(cmd.Stdout()).AddRef())
		h.Cmd.Stderr = output(ctx, iostream.Stream(cmd.Stderr()).AddRef())

		cherr <- h.Cmd.Start()
		return h.Cmd.Wait()
	})

	return <-cherr
}

func (h *handle) Signal(_ context.Context, call api.Unix_Proc_signal) (err error) {
	switch call.Args().Signal() {
	case api.Unix_Proc_Signal_sigINT:
		return h.Cmd.Process.Signal(syscall.SIGINT)

	case api.Unix_Proc_Signal_sigTERM:
		return h.Cmd.Process.Signal(syscall.SIGTERM)

	case api.Unix_Proc_Signal_sigKILL:
		return h.Cmd.Process.Signal(syscall.SIGKILL)

	default:
		return fmt.Errorf("unknown signal: %#x", int(call.Args().Signal()))
	}
}

func input(ctx context.Context, p iostream.Provider) io.Reader {
	if p.Client == (capnp.Client{}) {
		return nil
	}

	pr, pw := io.Pipe()
	go func() {
		f, release := p.Provide(ctx, iostream.New(pw, nil))
		defer release()

		if err := f.Await(ctx); err != nil {
			pw.CloseWithError(err)
		}
	}()

	return pr
}

func output(ctx context.Context, s iostream.Stream) io.Writer {
	if s.Client == (capnp.Client{}) {
		return nil
	}

	return s.Writer(ctx)
}

func stringSlice(load func() (capnp.TextList, error)) ([]string, error) {
	text, err := load()
	if err != nil {
		return nil, err
	}

	ss := make([]string, text.Len())
	for i := range ss {
		if ss[i], err = text.At(i); err != nil {
			break
		}
	}

	return ss, err
}
