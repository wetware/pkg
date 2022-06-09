package iostream

import (
	"bytes"
	"context"
	"io"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	chan_api "github.com/wetware/ww/internal/api/channel"
	"github.com/wetware/ww/internal/api/iostream"
	"github.com/wetware/ww/pkg/ocap"
	"github.com/wetware/ww/pkg/ocap/channel"
)

// Provider is the read end of a Unix byte-stream. It works by
// setting a StreamWriter as a callback, which is invoked whenever
// new data becomes available.
type Provider iostream.Provider

func (sr Provider) AddRef() Provider {
	return Provider{
		Client: sr.Client.AddRef(),
	}
}

func (sr Provider) Release() {
	sr.Client.Release()
}

// NewReader wraps r in a StreamReader and sets the supplied policy.
// If r implements io.Closer, it will be called automatically when
// the returned StreamReader shuts down.
func NewReader(r io.Reader, p *server.Policy) Provider {
	sr := &sreader{Reader: r}
	return Provider(iostream.Provider_ServerToClient(sr, p))
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
func (sr Provider) SetDst(ctx context.Context, sw Stream) (ocap.Future, capnp.ReleaseFunc) {
	stream := func(ps iostream.Provider_provide_Params) error {
		return ps.SetStream(iostream.Stream(sw))
	}

	f, release := iostream.Provider(sr).Provide(ctx, stream)
	return ocap.Future(f), release
}

// Stream is the write end of a Unix byte-stream.  It provides
// push semantics for transmitting streams of abitrary bytes.
// It is important to note that Stream MAY arbitrarily segment
// bytes.  Applications MAY implement their own framing.
type Stream iostream.Provider

func (sw Stream) AddRef() Stream {
	return Stream{
		Client: sw.Client.AddRef(),
	}
}

func (sw Stream) Release() {
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
func NewWriter(w io.Writer, p *server.Policy) Stream {
	sw := &swriter{Writer: w}
	return Stream(iostream.Stream_ServerToClient(sw, p))
}

// Write the bytes to the underlying stream.  Contrary to Go's io.Write,
// sw.Write will return after all bytes have been written to the stream,
// or an error occurs (whichever happens first).
func (sw Stream) Write(ctx context.Context, b []byte) (ocap.Future, capnp.ReleaseFunc) {
	f, release := iostream.Stream(sw).Send(ctx, channel.Data(b))
	return ocap.Future(f), release
}

// Close the underlying stream, signalling successful termination to any
// downstream consumers.  Close MUST be called when terminating, even if
// a previous write has failed.  We may relax this rule in the future.
func (sw Stream) Close(ctx context.Context) error {
	f, release := iostream.Stream(sw).Close(ctx, nil)
	defer release()

	_, err := f.Struct()
	return err
}

// Writer returns an io.Writer translates calls to its Write() method
// into calls to sw.Write().  The supplied context is implicitly passed
// to all sw.Write() calls.  Callers MAY implement per-write timeouts by
// repeatedly calling sw.Writer() with a fresh context.
func (sw Stream) Writer(ctx context.Context) io.Writer {
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

func (sr *sreader) Provide(ctx context.Context, call iostream.Provider_provide) (err error) {
	callback := Stream(call.Args().Stream())

	// stream terminated gracefully?
	if err = stream(callback.Writer(ctx), sr); err == nil {
		err = callback.Close(ctx)
	}

	return err
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
	if err == nil {
		// Don't close the underlying writer here.  Certain writer implementations,
		// such as net.Conn, may produce temporary errors.
		err = stream(sw, bytes.NewReader(ptr.Data()))
	}

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

func stream(w io.Writer, r io.Reader) error {
	_, err := io.Copy(w, r)
	return err
}

// writerFunc is a function type that implements io.Writer.
type writerFunc func([]byte) (int, error)

// Write calls the function with b as its argument.
func (write writerFunc) Write(b []byte) (int, error) {
	return write(b)
}
