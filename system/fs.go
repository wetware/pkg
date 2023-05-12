package system

import (
	"context"
	"errors"
	"io/fs"
	"net"
	"time"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/lthibault/log"
	"github.com/wetware/ww/pkg/anchor"
)

type FS struct {
	Ctx  context.Context
	Log  log.Logger
	Root *anchor.Node
}

// Open the named file.  Returns a *Socket.
func (fs FS) Open(name string) (fs.File, error) {
	host, guest := net.Pipe()
	go func() {
		// boot, resolver := capnp.NewLocalPromise[anchor.Anchor]()
		sock := &Socket{
			NS:   name,
			Log:  fs.Log.WithField("sock", "host"),
			Conn: host,
		}

		conn := rpc.NewConn(rpc.NewStreamTransport(sock), &rpc.Options{
			BootstrapClient: capnp.ErrorClient(errors.New("test")), // capnp.Client(fs.Root.Anchor()),
			ErrorReporter: errLogger{
				Logger: fs.Log.WithField("conn", "host"),
			},
		})
		defer conn.Close()

		// // FIXME:  this is a stub.  Add proper auth.
		// resolver.Fulfill(fs.Root.Anchor())

		select {
		case <-conn.Done(): // conn is closed by authenticate if auth fails
		case <-fs.Ctx.Done(): // close conn if the program is exiting
		}
	}()
	defer fs.Log.Debug("session started")

	return &Socket{
		NS:   name,
		Log:  fs.Log.WithField("sock", "guest"),
		Conn: guest,
	}, nil
}

type errLogger struct {
	log.Logger
}

func (e errLogger) ReportError(err error) {
	if err != nil {
		e.WithError(err).Warn("rpc connection failed")
	}
}

// Socket is a named connection that satisfies the fs.File interface.
type Socket struct {
	NS  string
	Log log.Logger
	net.Conn
}

func (sock *Socket) Read(b []byte) (n int, err error) {
	if n, err = sock.Conn.Read(b); err != nil {
		sock.Log.WithError(err).
			WithField("bytes", n).
			WithField("name", sock.NS).
			Info("error reading from socket")
	} else {
		sock.Log.
			WithField("bytes", n).
			WithField("name", sock.NS).
			Info("read data from socket")
	}

	return
}

func (sock *Socket) Write(b []byte) (n int, err error) {
	if n, err = sock.Conn.Write(b); err != nil {
		sock.Log.WithError(err).
			WithField("bytes", n).
			WithField("name", sock.NS).
			Info("error writing to socket")
	} else {
		sock.Log.
			WithField("bytes", n).
			WithField("name", sock.NS).
			Info("wrote data to socket")
	}

	return
}

func (sock *Socket) String() string {
	return sock.NS
}

func (sock *Socket) Stat() (fs.FileInfo, error) {
	return sock, nil
}

// base name of the file
func (sock *Socket) Name() string {
	return sock.NS
}

// length in bytes for regular files; system-dependent for others
func (sock *Socket) Size() int64 {
	return 0
}

// file mode bits
func (sock *Socket) Mode() fs.FileMode {
	return fs.ModeNamedPipe
}

// modification time
func (sock *Socket) ModTime() time.Time {
	return time.Now()
}

// abbreviation for Mode().IsDir()
func (sock *Socket) IsDir() bool {
	return false
}

// underlying data source (can return nil)
func (sock *Socket) Sys() any {
	return sock
}
