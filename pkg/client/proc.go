package client

import (
	"context"
	"runtime"

	"capnproto.org/go/capnp/v3"
	"github.com/wetware/ww/pkg/cap/proc"
	"github.com/wetware/ww/pkg/cap/proc/unix"
)

type UnixProc struct {
	unix.Client
}

func (p UnixProc) Exec(ctx context.Context, name string, args ...string) *Process {
	return newProcess(p.Client.Exec(ctx, name, args...))
}

type Process struct {
	client  proc.Process
	release capnp.ReleaseFunc
}

func newProcess(p proc.Process, release capnp.ReleaseFunc) *Process {
	proc := &Process{client: p, release: release}
	runtime.SetFinalizer(proc, release)
	return proc
}

func (c Process) Start(ctx context.Context) error {
	return c.client.Start(ctx)
}

func (c Process) Wait(ctx context.Context) error {
	err := c.client.Wait(ctx)
	c.release() // TODO: is this a correct beehavior?
	runtime.SetFinalizer(c, nil)
	return err
}

func (c *Process) StderrPipe(ctx context.Context) *ReadCloser {
	stderr, release := c.client.StderrPipe(ctx)
	rc := &ReadCloser{client: stderr}

	runtime.SetFinalizer(rc, release)
	return rc
}

func (c *Process) StdoutPipe(ctx context.Context) *ReadCloser {
	stdout, release := c.client.StdoutPipe(ctx)
	rc := &ReadCloser{client: stdout, release: release}

	runtime.SetFinalizer(rc, release)
	return rc
}

func (c *Process) StdinPipe(ctx context.Context) *WriteCloser {
	stdin, release := c.client.StdinPipe(ctx)
	rc := &WriteCloser{client: stdin, release: release}

	runtime.SetFinalizer(rc, release)
	return rc
}

type ReadCloser struct {
	client  proc.ReadCloser
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
	client  proc.WriteCloser
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
