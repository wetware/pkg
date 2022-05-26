package client

import (
	"context"
	"runtime"

	"capnproto.org/go/capnp/v3"
	"github.com/wetware/ww/pkg/cap/proc/unix"
)

type UnixExecutor struct {
	unix.Client
}

func (p UnixExecutor) Exec(ctx context.Context, name string, args ...string) *UnixProcess {
	client, release := p.Client.Exec(ctx, name, args...)
	proc := &UnixProcess{client: client, release: release}
	runtime.SetFinalizer(proc, release)
	return proc
}

type UnixProcess struct {
	client  unix.Process
	release capnp.ReleaseFunc
}

func (c UnixProcess) Start(ctx context.Context) error {
	return c.client.Start(ctx)
}

func (c UnixProcess) Wait(ctx context.Context) error {
	err := c.client.Wait(ctx)
	c.release() // TODO: is this a correct behavior?
	runtime.SetFinalizer(c, nil)
	return err
}

func (c *UnixProcess) StderrPipe(ctx context.Context) *ReadCloser {
	stderr, release := c.client.StderrPipe(ctx)
	rc := &ReadCloser{client: stderr}
	runtime.SetFinalizer(rc, release)
	return rc
}

func (c *UnixProcess) StdoutPipe(ctx context.Context) *ReadCloser {
	stdout, release := c.client.StdoutPipe(ctx)
	rc := &ReadCloser{client: stdout, release: release}
	runtime.SetFinalizer(rc, release)
	return rc
}

func (c *UnixProcess) StdinPipe(ctx context.Context) *WriteCloser {
	stdin, release := c.client.StdinPipe(ctx)
	rc := &WriteCloser{client: stdin, release: release}
	runtime.SetFinalizer(rc, release)
	return rc
}

type ReadCloser struct {
	client  unix.ReadCloser
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
	client  unix.WriteCloser
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
