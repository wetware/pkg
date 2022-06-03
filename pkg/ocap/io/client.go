package io

import (
	"context"
	"io"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	api "github.com/wetware/ww/internal/api/io"
	"github.com/wetware/ww/pkg/ocap"
)

type Closer api.Closer

func NewCloser(c io.Closer, p *server.Policy) Closer {
	return NewCloseClient(CloseServer{Closer: c}, p)
}

func NewCloseClient(c Closeable, p *server.Policy) Closer {
	return Closer(api.Closer_ServerToClient(c, p))
}

func (c Closer) AddRef() Closer {
	return Closer{
		Client: c.Client.AddRef(),
	}
}

func (c Closer) Release() {
	c.Client.Release()
}

func (c Closer) Close(ctx context.Context) error {
	f, release := api.Closer(c).Close(ctx, nil)
	defer release()

	return ocap.Future(f).Await(ctx)
}

type Reader api.Reader

func NewReader(r io.Reader, p *server.Policy) Reader {
	return NewReadClient(ReadServer{Reader: r}, p)
}

func NewReadClient(r Readable, p *server.Policy) Reader {
	return Reader(api.Reader_ServerToClient(r, p))
}

func (r Reader) AddRef() Reader {
	return Reader{
		Client: r.Client.AddRef(),
	}
}

func (r Reader) Release() {
	r.Client.Release()
}

func (r Reader) Read(ctx context.Context, n int) (FutureRead, capnp.ReleaseFunc) {
	f, release := api.Reader(r).Read(ctx, func(ps api.Reader_read_Params) error {
		ps.SetN(uint16(n))
		return nil
	})

	return FutureRead(f), release
}

type Writer api.Writer

func NewWriter(w io.Writer, p *server.Policy) Writer {
	return NewWriteClient(WriteServer{Writer: w}, p)
}

func NewWriteClient(w Writeable, p *server.Policy) Writer {
	return Writer(api.Writer_ServerToClient(w, p))
}

func (w Writer) AddRef() Writer {
	return Writer{
		Client: w.Client.AddRef(),
	}
}

func (w Writer) Release() {
	w.Client.Release()
}

func (w Writer) Write(ctx context.Context, b []byte) (FutureWrite, capnp.ReleaseFunc) {
	f, release := api.Writer(w).Write(ctx, func(ps api.Writer_write_Params) error {
		return ps.SetData(b)
	})

	return FutureWrite(f), release
}

type ReadCloser api.ReadCloser

func NewReadCloser(rc io.ReadCloser, p *server.Policy) ReadCloser {
	return NewReadCloseClient(ReadCloseServer{ReadCloser: rc}, p)
}

func NewReadCloseClient(rc ReadCloseable, p *server.Policy) ReadCloser {
	return ReadCloser(api.ReadCloser_ServerToClient(rc, p))
}

func (rc ReadCloser) AddRef() ReadCloser {
	return ReadCloser{
		Client: rc.Client.AddRef(),
	}
}

func (rc ReadCloser) Release() {
	rc.Client.Release()
}

func (rc ReadCloser) Close(ctx context.Context) error {
	return Closer(rc).Close(ctx)
}

func (rc ReadCloser) Read(ctx context.Context, n int) (FutureRead, capnp.ReleaseFunc) {
	return Reader(rc).Read(ctx, n)
}

type WriteCloser api.WriteCloser

func NewWriteCloser(wc io.WriteCloser, p *server.Policy) WriteCloser {
	return NewWriteCloseClient(WriteCloseServer{WriteCloser: wc}, p)
}

func NewWriteCloseClient(wc WriteCloseable, p *server.Policy) WriteCloser {
	return WriteCloser(api.WriteCloser_ServerToClient(wc, p))
}

func (wc WriteCloser) AddRef() WriteCloser {
	return WriteCloser{
		Client: wc.Client.AddRef(),
	}
}

func (wc WriteCloser) Release() {
	wc.Client.Release()
}

func (wc WriteCloser) Write(ctx context.Context, b []byte) (FutureWrite, capnp.ReleaseFunc) {
	return Writer(wc).Write(ctx, b)
}

func (wc WriteCloser) Close(ctx context.Context) error {
	return Closer(wc).Close(ctx)
}

type ReadWriter api.ReadWriter

func NewReadWriter(rw io.ReadWriter, p *server.Policy) ReadWriter {
	return NewReadWriteClient(ReadWriteServer{ReadWriter: rw}, p)
}

func NewReadWriteClient(rw ReadWriteable, p *server.Policy) ReadWriter {
	return ReadWriter(api.ReadWriter_ServerToClient(rw, p))
}

func (rw ReadWriter) AddRef() ReadWriter {
	return ReadWriter{
		Client: rw.Client.AddRef(),
	}
}

func (rw ReadWriter) Release() {
	rw.Client.Release()
}

func (rw ReadWriter) Read(ctx context.Context, n int) (FutureRead, capnp.ReleaseFunc) {
	return Reader(rw).Read(ctx, n)
}

func (rw ReadWriter) Write(ctx context.Context, b []byte) (FutureWrite, capnp.ReleaseFunc) {
	return Writer(rw).Write(ctx, b)
}

type ReadWriteCloser api.ReadWriteCloser

func NewReadWriteCloser(rwc io.ReadWriteCloser, p *server.Policy) ReadWriteCloser {
	return NewReadWriteCloseClient(ReadWriteCloseServer{ReadWriteCloser: rwc}, p)
}

func NewReadWriteCloseClient(rwc ReadWriteCloseable, p *server.Policy) ReadWriteCloser {
	return ReadWriteCloser(api.ReadWriteCloser_ServerToClient(rwc, p))
}

func (rwc ReadWriteCloser) AddRef() ReadWriteCloser {
	return ReadWriteCloser{
		Client: rwc.Client.AddRef(),
	}
}

func (rwc ReadWriteCloser) Release() {
	rwc.Client.Release()
}

func (rwc ReadWriteCloser) Read(ctx context.Context, n int) (FutureRead, capnp.ReleaseFunc) {
	return Reader(rwc).Read(ctx, n)
}

func (rwc ReadWriteCloser) Write(ctx context.Context, b []byte) (FutureWrite, capnp.ReleaseFunc) {
	return Writer(rwc).Write(ctx, b)
}

func (rwc ReadWriteCloser) Close(ctx context.Context) error {
	return Closer(rwc).Close(ctx)
}

/*
	Typed Futures
*/

type FutureRead api.Reader_read_Results_Future

func (f FutureRead) Err() error {
	_, err := f.Struct()
	return err
}

func (f FutureRead) Await(ctx context.Context) ([]byte, error) {
	select {
	case <-f.Done():
		return f.Bytes()

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (f FutureRead) Bytes() ([]byte, error) {
	res, err := api.Reader_read_Results_Future(f).Struct()
	if err != nil {
		return nil, err
	}

	b, err := res.Data()
	if err != nil {
		return nil, err
	}

	e, err := res.Err()
	if err != nil {
		return nil, err
	}

	return b, ioError(e).Err()
}

type FutureWrite api.Writer_write_Results_Future

func (f FutureWrite) Err() error {
	_, err := f.Struct()
	return err
}

func (f FutureWrite) Await(ctx context.Context) (int64, error) {
	select {
	case <-f.Done():
		return f.N()

	case <-ctx.Done():
		return -1, ctx.Err()
	}
}

func (f FutureWrite) N() (int64, error) {
	res, err := api.Writer_write_Results_Future(f).Struct()
	if err != nil {
		return -1, err
	}

	e, err := res.Err()
	if err == nil {
		err = ioError(e).Err()
	}

	return res.N(), err
}
