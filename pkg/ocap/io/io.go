package io

import (
	"context"

	api "github.com/wetware/ww/internal/api/io"
)

type (
	MethodClose = api.Closer_close
	MethodRead  = api.Reader_read
	MethodWrite = api.Writer_write
)

// Readable types implement the Reader capability.
type Readable interface {
	Read(context.Context, MethodRead) error
}

// Writeable types implement the Writer capability.
type Writeable interface {
	Write(context.Context, MethodWrite) error
}

// Closeable types implement the Closer capability.
type Closeable interface {
	Close(context.Context, MethodClose) error
}

// ReadClosable types implement the ReadCloser capability.
type ReadCloseable interface {
	Readable
	Closeable
}

// WriteCloseable types implement the WriteCloser capability.
type WriteCloseable interface {
	Writeable
	Closeable
}

// ReadWriteable types implement the ReadWriter capability
type ReadWriteable interface {
	Readable
	Writeable
}

// ReadWriteCloseable types implement the ReadWriteCloser capability.
type ReadWriteCloseable interface {
	ReadWriteable
	Closeable
}
