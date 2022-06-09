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
	"github.com/wetware/ww/pkg/ocap"
	"github.com/wetware/ww/pkg/ocap/process"
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
	return process.P(p).Wait(ctx)
}

func (p Proc) Signal(ctx context.Context, s syscall.Signal) (ocap.Future, capnp.ReleaseFunc) {
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

	return ocap.Future(f), release
}

// cmdServer is the server implementation of UnixProc
type cmdServer struct {
	*exec.Cmd
	cancel context.CancelFunc
	done   chan struct{}
	err    error
}

func (c *cmdServer) Shutdown() {
	defer func() {
		c.cancel() // SIGKILL
		<-c.done   // wait for process to terminate
	}()

	// Attempt to gracefully shut down the process, but kill it if the
	// timeout is exceeded.
	if err := c.Process.Signal(syscall.SIGTERM); err == nil {
		select {
		case <-c.done:
		case <-time.After(time.Second * 5):
		}
	}
}

func newCommandServer(ctx context.Context, param api.Unix_Command) (*cmdServer, error) {
	path, err := param.Path()
	if err != nil {
		return nil, err
	}

	args, err := stringSlice(param.Args)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, path, args...)

	if cmd.Env, err = stringSlice(param.Env); err == nil {
		cmd.Stdin = input(ctx, param.Stdin().AddRef())
		cmd.Stdout = output(ctx, param.Stdout().AddRef())
		cmd.Stderr = output(ctx, param.Stderr().AddRef())
	}

	return &cmdServer{
		Cmd:    cmd,
		cancel: cancel,
		done:   make(chan struct{}),
	}, err
}

func (c *cmdServer) Start() (err error) {
	if err = c.Cmd.Start(); err == nil {
		go func() {
			defer close(c.done)
			c.err = c.Cmd.Wait()
		}()
	}

	return err

}

func (c *cmdServer) Wait(ctx context.Context, _ api.P_wait) error {
	select {
	case <-c.done:
		return c.err

	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *cmdServer) Signal(_ context.Context, call api.Unix_Proc_signal) error {
	switch call.Args().Signal() {
	case api.Unix_Proc_Signal_sigINT:
		return c.Process.Signal(syscall.SIGINT)

	case api.Unix_Proc_Signal_sigTERM:
		return c.Process.Signal(syscall.SIGTERM)

	case api.Unix_Proc_Signal_sigKILL:
		return c.Process.Signal(syscall.SIGKILL)

	default:
		return fmt.Errorf("unknown signal: %#x", int(call.Args().Signal()))
	}
}

func input(ctx context.Context, reader api.Unix_StreamReader) io.Reader {
	if reader.Client == nil {
		return nil
	}

	pr, pw := io.Pipe()
	go func() {
		f, release := StreamReader(reader).SetDst(ctx, NewWriter(pw, nil))
		defer release()

		if err := f.Await(ctx); err != nil {
			pw.CloseWithError(err)
		}
	}()

	return pr
}

func output(ctx context.Context, writer api.Unix_StreamWriter) io.Writer {
	if writer.Client == nil {
		return nil
	}

	return StreamWriter(writer).Writer(ctx)
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
