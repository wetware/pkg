package unix

import (
	"context"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/ww/internal/api/proc"
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

		return p.SetCommand(cmd.ToPtr())
	}
}

type Client struct {
	client api.Executor
}

func (c Client) Exec(ctx context.Context, name string, args ...string) (Process, capnp.ReleaseFunc) {
	fut, release := c.client.Exec(ctx, unixCmd(name, args...))
	return &ProcessClient{client: fut.Proc()}, release
}

type ProcessClient struct {
	client api.Process
}

func (c ProcessClient) Start(ctx context.Context) error {
	fut, release := c.client.Start(ctx, nil)
	defer release()

	_, err := fut.Struct()
	return err
}

func (c ProcessClient) Wait(ctx context.Context) error {
	fut, release := c.client.Wait(ctx, nil)
	defer release()

	_, err := fut.Struct()
	return err
}

func (c ProcessClient) StderrPipe(ctx context.Context) (ReadCloser, capnp.ReleaseFunc) {
	fut, release := c.client.StderrPipe(ctx, nil)

	return &ReadCloserClient{client: fut.Rc()}, release
}

func (c *ProcessClient) StdoutPipe(ctx context.Context) (ReadCloser, capnp.ReleaseFunc) {
	fut, release := c.client.StdoutPipe(ctx, nil)

	return &ReadCloserClient{client: fut.Rc()}, release
}

func (c ProcessClient) StdinPipe(ctx context.Context) (WriteCloser, capnp.ReleaseFunc) {
	fut, release := c.client.StdinPipe(ctx, nil)

	return &WriteCloserClient{client: fut.Wc()}, release
}

type ReadCloserClient struct {
	client api.ReadCloser
}

func (c ReadCloserClient) Read(ctx context.Context, b []byte) (n int, err error) {
	fut, release := c.client.Read(ctx, func(p api.Reader_read_Params) error {
		p.SetN(int64(len(b)))
		return nil
	})
	defer release()

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
}

func (c ReadCloserClient) Close(ctx context.Context) error {
	fut, release := c.client.Close(ctx, nil)
	defer release()

	_, err := fut.Struct()
	return err
}

type WriteCloserClient struct {
	client api.WriteCloser
}

func (c WriteCloserClient) Write(ctx context.Context, b []byte) (n int, err error) {
	fut, release := c.client.Write(ctx, func(p api.Writer_write_Params) error {
		return p.SetData(b)
	})
	defer release()

	results, err := fut.Struct()
	if err != nil {
		return 0, err
	}
	return int(results.N()), nil
}

func (c WriteCloserClient) Close(ctx context.Context) error {
	fut, release := c.client.Close(ctx, nil)
	defer release()

	_, err := fut.Struct()
	return err
}
