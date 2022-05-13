package client

import (
	"context"
	"runtime"

	"capnproto.org/go/capnp/v3"
	"github.com/wetware/ww/pkg/cap/proc"
)

type Proc struct {
	Client proc.Client
}

func (p *Proc) Command(ctx context.Context, name string, args ...string) *Cmd {
	capCmd, release := p.Client.Command(ctx, name, args...)
	cmd := &Cmd{client: capCmd}

	runtime.SetFinalizer(cmd, release)

	return cmd
}

type Cmd struct {
	client  *proc.CmdClient
	release capnp.ReleaseFunc
}

func (c *Cmd) Start(ctx context.Context) error {
	return c.client.Start(ctx)
}

func (c *Cmd) Wait(ctx context.Context) error {
	err := c.client.Wait(ctx)
	c.release() // TODO: is this a correct beehavior?
	return err
}

func (c *Cmd) StderrPipe(ctx context.Context) *ReadCloser {
	stderr, release := c.client.StderrPipe(ctx)
	rc := &ReadCloser{client: stderr}

	runtime.SetFinalizer(rc, release)
	return rc
}

func (c *Cmd) StdoutPipe(ctx context.Context) *ReadCloser {
	stdout, release := c.client.StdoutPipe(ctx)
	rc := &ReadCloser{client: stdout, release: release}

	runtime.SetFinalizer(rc, release)
	return rc
}

func (c *Cmd) StdinPipe(ctx context.Context) *WriteCloser {
	stdin, release := c.client.StdinPipe(ctx)
	rc := &WriteCloser{client: stdin, release: release}

	runtime.SetFinalizer(rc, release)
	return rc
}

type ReadCloser struct {
	client  *proc.ReadCloserClient
	release capnp.ReleaseFunc
}

func (rc *ReadCloser) Read(ctx context.Context, p []byte) (n int, err error) {
	return rc.client.Read(ctx, p)
}

func (rc *ReadCloser) Close(ctx context.Context) (err error) {
	err = rc.client.Close(ctx)
	rc.release()
	runtime.SetFinalizer(rc, nil)
	return
}

type WriteCloser struct {
	client  *proc.WriteCloserClient
	release capnp.ReleaseFunc
}

func (wc *WriteCloser) Write(ctx context.Context, p []byte) (n int, err error) {
	return wc.client.Write(ctx, p)
}

func (wc *WriteCloser) Close(ctx context.Context) (err error) {
	err = wc.client.Close(ctx)
	wc.release()
	runtime.SetFinalizer(wc, nil)
	return
}
