package unix

import (
	"bytes"
	"context"
	"io"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	chan_api "github.com/wetware/ww/internal/api/channel"
	api "github.com/wetware/ww/internal/api/proc"
	"github.com/wetware/ww/pkg/ocap"
	"github.com/wetware/ww/pkg/ocap/channel"
)

// StreamReader is the read end of a Unix byte-stream. It works by
// setting a StreamWriter as a callback, which is invoked whenever
// new data becomes available.
type StreamReader api.Unix_StreamReader

func (sr StreamReader) AddRef() StreamReader {
	return StreamReader{
		Client: sr.Client.AddRef(),
	}
}

func (sr StreamReader) Release() {
	sr.Client.Release()
}

// NewReader wraps r in a StreamReader and sets the supplied policy.
// If r implements io.Closer, it will be called automatically when
// the returned StreamReader shuts down.
func NewReader(r io.Reader, p *server.Policy) StreamReader {
	sr := &sreader{Reader: r}
	return StreamReader(api.Unix_StreamReader_ServerToClient(sr, p))
}

// SetDst assigns the supplied StreamWriter as the destination for
// incoming bytes in the stream.  The dst parameter is effectively
// a callback.  Only the first call to SetDst will be honored, and
// any subsequent calls will return an error.  This includes calls
// made by remote vats.
//
// Calling SetWriter transfers ownership of w to sr, and w will be
// closed when sr is closed, or the underlying stream fails.  Thus,
// one SHOULD NOT call any of w's methods after SetWriter returns.
//
// Callers SHOULD enforce the following invariant on dst:  after a
// call to SetDst returns, all references to dst are owned by sr.
// In practice, callers MAY relax this invariant when either:
//
//   (a)  References not owned by sr are released before the future
//        returned by SetDst() is resolved.
//
//	 (b)  The consumer behind 'dst' does not distinguish between
//        normal and erroneous stream termination.
//
func (sr StreamReader) SetDst(ctx context.Context, dst StreamWriter) (ocap.Future, capnp.ReleaseFunc) {
	f, release := api.Unix_StreamReader(sr).SetDst(ctx, func(ps api.Unix_StreamReader_setDst_Params) error {
		return ps.SetDst(api.Unix_StreamWriter(dst))
	})

	return ocap.Future(f), release
}

// StreamWriter is the write end of a Unix byte-stream.  It provides
// push semantics for transmitting streams of abitrary bytes.  It is
// important to note that StreamWriter MAY arbitrarily segment bytes.
// Applications MAY implement their own framing.
type StreamWriter api.Unix_StreamWriter

func (sw StreamWriter) AddRef() StreamWriter {
	return StreamWriter{
		Client: sw.Client.AddRef(),
	}
}

func (sw StreamWriter) Release() {
	sw.Client.Release()
}

// NewWriter wraps the supplied WriteCloser in a StreamWriter.
// Callers MUST call the returned StreamWriter's Close() method
// before releasing the client, to signal graceful termination.
// If the StreamWriter is released before a call to Close returns,
// the downstream consumer SHALL interpret this as ErrUnexpectedEOF.
//
// If w implements io.Closer, it will be closed before the call to
// StreamWriter.Close() resolves, or after the last client reference
// is released, whichever comes first.
func NewWriter(w io.Writer, p *server.Policy) StreamWriter {
	sw := &swriter{Writer: w}
	return StreamWriter(api.Unix_StreamWriter_ServerToClient(sw, p))
}

// Write the bytes to the underlying stream.  Contrary to Go's io.Write,
// sw.Write will return after all bytes have been written to the stream,
// or an error occurs (whichever happens first).
func (sw StreamWriter) Write(ctx context.Context, b []byte) (ocap.Future, capnp.ReleaseFunc) {
	f, release := api.Unix_StreamWriter(sw).Send(ctx, channel.Data(b))
	return ocap.Future(f), release
}

// Close the underlying stream, signalling successful termination to any
// downstream consumers.  Close MUST be called when terminating, even if
// a previous write has failed.  We may relax this rule in the future.
func (sw StreamWriter) Close(ctx context.Context) error {
	f, release := api.Unix_StreamWriter(sw).Close(ctx, nil)
	defer release()

	_, err := f.Struct()
	return err
}

// Writer returns an io.Writer translates calls to its Write() method
// into calls to sw.Write().  The supplied context is implicitly passed
// to all sw.Write() calls.  Callers MAY implement per-write timeouts by
// repeatedly calling sw.Writer() with a fresh context.
func (sw StreamWriter) Writer(ctx context.Context) io.Writer {
	return writerFunc(func(b []byte) (int, error) {
		f, release := sw.Write(ctx, b)
		defer release()

		return len(b), f.Await(ctx)
	})
}

/*
	Server implementations
*/

// sreader is the server type for StreamReader.
type sreader struct {
	io.Reader
}

func (sr *sreader) Shutdown() {
	if c, ok := sr.Reader.(io.Closer); ok {
		_ = c.Close()
	}
}

func (sr *sreader) SetDst(ctx context.Context, call api.Unix_StreamReader_setDst) error {
	callback := StreamWriter(call.Args().Dst())

	if _, err := io.Copy(callback.Writer(ctx), sr); err != nil {
		return err
	}

	// Stream terminated gracefully.  Signal close.
	return callback.Close(ctx)
}

// swriter is the server type for StreamWriter.  It wraps an io.Closer and
// exports a Send method, thereby satisfying the channel.Sender capability
// interface.
type swriter struct {
	io.Writer
}

func (sw *swriter) Shutdown() { _ = sw.close() }

func (sw *swriter) Send(_ context.Context, call chan_api.Sender_send) error {
	ptr, err := call.Args().Value()
	if err != nil {
		return err
	}

	// Don't close the underlying writer here.  Certain writer implementations,
	// such as net.Conn, may produce temporary errors.
	_, err = io.Copy(sw, bytes.NewReader(ptr.Data()))
	return err
}

func (sw *swriter) Close(context.Context, chan_api.Closer_close) error {
	return sw.close()
}

func (sw *swriter) close() (err error) {
	if c, ok := sw.Writer.(io.Closer); ok {
		err = c.Close()
		sw.Writer = nil // DEFENSIVE: prevent writer from being closed twice
	}

	return
}

// writerFunc is a function type that implements io.Writer.
type writerFunc func([]byte) (int, error)

// Write calls the function with b as its argument.
func (write writerFunc) Write(b []byte) (int, error) {
	return write(b)
}
