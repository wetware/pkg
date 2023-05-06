package fs

import (
	"errors"
	"io/fs"
	"math"
	"net"
	"time"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
)

type FS struct {
	Guest, Host     net.Conn
	BootstrapClient capnp.Client
}

func (fs FS) Open(name string) (fs.File, error) {
	if fs.Guest == nil || fs.Host == nil {
		return File{}, errors.New("TODO")
	}

	go func() {
		conn := rpc.NewConn(
			rpc.NewStreamTransport(fs.Host),
			&rpc.Options{BootstrapClient: fs.BootstrapClient},
		)
		<-conn.Done()
	}()

	return File{
		conn: fs.Guest,
		name: name,
	}, nil
}

type File struct {
	name string
	conn net.Conn
}

func (f File) Stat() (fs.FileInfo, error) {
	return FileInfo{
		name: f.name,
	}, nil
}

func (f File) Read(b []byte) (int, error) {
	return f.conn.Read(b)
}

func (f File) Close() error {
	return f.conn.Close()
}

func (f File) Conn() net.Conn {
	return f.conn
}

type FileInfo struct {
	name string
}

func (fi FileInfo) Name() string {
	return fi.name
}

func (fi FileInfo) Size() int64 {
	return math.MinInt64 // TODO
}

func (fi FileInfo) Mode() fs.FileMode {
	return fs.ModeSocket
}

func (fi FileInfo) ModTime() time.Time {
	return time.Time{}
}

func (fi FileInfo) IsDir() bool {
	return false
}

func (fi FileInfo) Sys() any {
	return nil
}
