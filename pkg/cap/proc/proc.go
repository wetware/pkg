package proc

import (
	"context"

	"capnproto.org/go/capnp/v3"
)

type Process interface {
	Start(context.Context) error
	Wait(ctx context.Context) error
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
