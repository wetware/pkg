package proc

import (
	"context"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/ww/internal/api/proc"
)

type Client struct {
	client api.UnixExecutor
}

func (c *Client) Command(ctx context.Context, name string, args ...string) (*CmdClient, capnp.ReleaseFunc) {
	fut, release := c.client.Command(ctx, func(p api.UnixExecutor_command_Params) error {
		if err := p.SetName(name); err != nil {
			return err
		}

		arg, err := p.NewArg(int32(len(args)))
		if err != nil {
			return err
		}

		for i, a := range args {
			if err := arg.Set(i, a); err != nil {
				return err
			}
		}
		return nil
	})

	return &CmdClient{client: fut.Cmd()}, release
}

type CmdClient struct {
	client api.Cmd
}

func (c *CmdClient) Start(ctx context.Context) error {
	fut, release := c.client.Start(ctx, func(p api.Cmd_start_Params) error {
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

func (c *CmdClient) Wait(ctx context.Context) error {
	fut, release := c.client.Wait(ctx, func(p api.Cmd_wait_Params) error {
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

func (c *CmdClient) StderrPipe(ctx context.Context) (*ReadCloserClient, capnp.ReleaseFunc) {
	fut, release := c.client.StderrPipe(ctx, func(c api.Cmd_stderrPipe_Params) error {
		return nil
	})

	return &ReadCloserClient{client: fut.Rc()}, release
}

func (c *CmdClient) StdoutPipe(ctx context.Context) (*ReadCloserClient, capnp.ReleaseFunc) {
	fut, release := c.client.StdoutPipe(ctx, func(p api.Cmd_stdoutPipe_Params) error {
		return nil
	})

	return &ReadCloserClient{client: fut.Rc()}, release
}

func (c *CmdClient) StdinPipe(ctx context.Context) (*WriteCloserClient, capnp.ReleaseFunc) {
	fut, release := c.client.StdinPipe(ctx, func(p api.Cmd_stdinPipe_Params) error {
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
