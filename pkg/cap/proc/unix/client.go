package unix

import (
	"context"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/ww/internal/api/proc"
	"github.com/wetware/ww/pkg/cap/proc"
)

func unixCmd(name string, args ...string) func(p api.Executor_exec_Params) error {
	return func(p api.Executor_exec_Params) error {
		_, s, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			return err
		}

		cmd, err := api.NewRootUnixCommand(s)
		if err != nil {
			return err
		}

		if err := cmd.SetName(name); err != nil {
			return err
		}

		arg, err := cmd.NewArg(int32(len(args)))
		if err != nil {
			return err
		}

		for i, a := range args {
			if err := arg.Set(i, a); err != nil {
				return err
			}
		}

		return p.SetProfile(cmd.ToPtr())
	}
}

type Client struct {
	client api.Executor
}

func (c *Client) Exec(ctx context.Context, name string, args ...string) (proc.Process, capnp.ReleaseFunc) {
	fut, release := c.client.Exec(ctx, unixCmd(name, args...))
	return &ProcessClient{client: fut.Proc()}, release
}

type ProcessClient struct {
	client api.Process
}

func (c *ProcessClient) Start(ctx context.Context) error {
	fut, release := c.client.Start(ctx, func(p api.Process_start_Params) error {
		return nil
	})
	defer release()

	select {
	case <-fut.Done():
		_, err := fut.Struct()
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *ProcessClient) Wait(ctx context.Context) error {
	fut, release := c.client.Wait(ctx, func(p api.Process_wait_Params) error {
		return nil
	})
	defer release()

	select {
	case <-fut.Done():
		_, err := fut.Struct()
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *ProcessClient) StderrPipe(ctx context.Context) (proc.ReadCloser, capnp.ReleaseFunc) {
	fut, release := c.client.StderrPipe(ctx, func(c api.Process_stderrPipe_Params) error {
		return nil
	})

	return &ReadCloserClient{client: fut.Rc()}, release
}

func (c *ProcessClient) StdoutPipe(ctx context.Context) (proc.ReadCloser, capnp.ReleaseFunc) {
	fut, release := c.client.StdoutPipe(ctx, func(p api.Process_stdoutPipe_Params) error {
		return nil
	})

	return &ReadCloserClient{client: fut.Rc()}, release
}

func (c *ProcessClient) StdinPipe(ctx context.Context) (proc.WriteCloser, capnp.ReleaseFunc) {
	fut, release := c.client.StdinPipe(ctx, func(p api.Process_stdinPipe_Params) error {
		return nil
	})

	return &WriteCloserClient{client: fut.Wc()}, release
}

type ReadCloserClient struct {
	client api.ReadCloser
}

func (c *ReadCloserClient) Read(ctx context.Context, b []byte) (n int, err error) {
	fut, release := c.client.Read(ctx, func(p api.Reader_read_Params) error {
		p.SetN(int64(len(b)))
		return nil
	})
	defer release()

	select {
	case <-fut.Done():
		results, err := fut.Struct()
		if err != nil {
			return 0, err
		}
		data, err := results.Data()
		if err != nil {
			return 0, err
		}
		copy(b, data)

		return int(results.N()), nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func (c *ReadCloserClient) Close(ctx context.Context) error {
	fut, release := c.client.Close(ctx, func(c api.Closer_close_Params) error {
		return nil
	})
	defer release()

	select {
	case <-fut.Done():
		_, err := fut.Struct()
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

type WriteCloserClient struct {
	client api.WriteCloser
}

func (c *WriteCloserClient) Write(ctx context.Context, b []byte) (n int, err error) {
	fut, release := c.client.Write(ctx, func(p api.Writer_write_Params) error {
		return p.SetData(b)
	})
	defer release()

	select {
	case <-fut.Done():
		results, err := fut.Struct()
		if err != nil {
			return 0, err
		}
		return int(results.N()), nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func (c *WriteCloserClient) Close(ctx context.Context) error {
	fut, release := c.client.Close(ctx, func(c api.Closer_close_Params) error {
		return nil
	})
	defer release()

	select {
	case <-fut.Done():
		_, err := fut.Struct()
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
