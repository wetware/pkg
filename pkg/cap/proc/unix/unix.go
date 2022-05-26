package unix

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"github.com/wetware/ww/pkg/cap/proc"
)

type Process interface {
	proc.Process
	StderrPipe(ctx context.Context) (ReadCloser, capnp.ReleaseFunc)
	StdoutPipe(ctx context.Context) (ReadCloser, capnp.ReleaseFunc)
	StdinPipe(ctx context.Context) (WriteCloser, capnp.ReleaseFunc)
}

type ReadCloser interface {
	Reader
	Closer
}

type WriteCloser interface {
	Writer
	Closer
}

type Reader interface {
	Read(ctx context.Context, b []byte) (n int, err error)
}

type Writer interface {
	Write(ctx context.Context, b []byte) (n int, err error)
}

type Closer interface {
	Close(ctx context.Context) error
}
