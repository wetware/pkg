package io

import (
	"context"
	"io"
)

type CloseServer struct {
	io.Closer
}

func (c CloseServer) Close(ctx context.Context, call MethodClose) error {
	return c.Closer.Close()
}

type ReadServer struct {
	io.Reader
}

func (r ReadServer) Read(ctx context.Context, call MethodRead) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	// Pre-allocate this, so that we don't paint ourselves into a corner
	// with an allocation failure *after* we've already read. This would
	// prevent us from delivering any bytes read from r.
	e, err := res.NewErr()
	if err != nil {
		return err
	}

	// N is limited to 64kb, which prevents resource-exhaustion attacks.
	// Callers can, and generally SHOULD, use capnp's flow control API
	// to ensure good performance.
	buf := make([]byte, call.Args().N())

	// Don't immediately error out!  Conformant to the io.Reader contract,
	// we need to support partial reads, even in the presence of errors.
	n, err := r.Reader.Read(buf)
	if err != nil {
		// must succeed; we already read data
		_ = ioError(e).Set(err)
	}

	return res.SetData(buf[:n])
}

type WriteServer struct {
	io.Writer
}

func (w WriteServer) Write(ctx context.Context, call MethodWrite) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	// Pre-allocate this, so that we don't paint ourselves into a corner
	// with an allocation failure *after* we've already written. This would
	// prevent us from reporting the number of byes written to w.
	e, err := res.NewErr()
	if err != nil {
		return err
	}

	b, err := call.Args().Data()
	if err != nil {
		return err
	}

	n, err := w.Writer.Write(b)
	if err != nil {
		// must succeed; we already written data
		_ = ioError(e).Set(err)
	}

	res.SetN(int64(n))
	return nil
}

type ReadCloseServer struct {
	io.ReadCloser
}

func (rc ReadCloseServer) Read(ctx context.Context, call MethodRead) error {
	return ReadServer{Reader: rc.ReadCloser}.Read(ctx, call)
}

func (rc ReadCloseServer) Close(ctx context.Context, call MethodClose) error {
	return CloseServer{Closer: rc.ReadCloser}.Close(ctx, call)
}

type WriteCloseServer struct {
	io.WriteCloser
}

func (w WriteCloseServer) Write(ctx context.Context, call MethodWrite) error {
	return WriteServer{Writer: w.WriteCloser}.Write(ctx, call)
}

func (w WriteCloseServer) Close(ctx context.Context, call MethodClose) error {
	return CloseServer{Closer: w.WriteCloser}.Close(ctx, call)
}

type ReadWriteServer struct {
	io.ReadWriter
}

func (rw ReadWriteServer) Write(ctx context.Context, call MethodWrite) error {
	return WriteServer{Writer: rw.ReadWriter}.Write(ctx, call)
}

func (rw ReadWriteServer) Read(ctx context.Context, call MethodRead) error {
	return ReadServer{Reader: rw.ReadWriter}.Read(ctx, call)
}

type ReadWriteCloseServer struct {
	io.ReadWriteCloser
}

func (rwc ReadWriteCloseServer) Write(ctx context.Context, call MethodWrite) error {
	return WriteServer{Writer: rwc.ReadWriteCloser}.Write(ctx, call)
}

func (rwc ReadWriteCloseServer) Read(ctx context.Context, call MethodRead) error {
	return ReadServer{Reader: rwc.ReadWriteCloser}.Read(ctx, call)
}

func (rwc ReadWriteCloseServer) Close(ctx context.Context, call MethodClose) error {
	return CloseServer{Closer: rwc.ReadWriteCloser}.Close(ctx, call)
}
