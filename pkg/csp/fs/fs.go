package fs

import (
	"errors"
	"io"
	"io/fs"
	"time"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
)

type FS struct {
	Guest, Host     io.ReadWriteCloser
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
		pipeEnd: fs.Guest,
		name:    name,
	}, nil
}

type File struct {
	name    string
	pipeEnd io.ReadWriteCloser
}

func (f File) Stat() (fs.FileInfo, error) {
	return FileInfo{
		name: f.name,
	}, nil
}

func (f File) Read(b []byte) (int, error) {
	return f.pipeEnd.Read(b)
}

func (f File) Close() error {
	return f.pipeEnd.Close()
}

func (f File) PipeEnd() io.ReadWriteCloser {
	return f.pipeEnd
}

type FileInfo struct {
	name string
}

func (fi FileInfo) Name() string {
	return fi.name
}

func (fi FileInfo) Size() int64 {
	return 1
	// return math.MinInt64 // TODO
}

func (fi FileInfo) Mode() fs.FileMode {
	return fs.ModeDir
	// return fs.ModeAppend
	// return fs.ModeSocket
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
