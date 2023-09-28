package system

import (
	"io"

	"github.com/lthibault/iopipes"
	"go.uber.org/multierr"
)

func Pipe() (io.ReadWriteCloser, io.ReadWriteCloser) {
	// TODO:  basic flow control using iopipes.DrainingPipe.
	p1r, p1w := iopipes.InfinitePipe()
	p2r, p2w := iopipes.InfinitePipe()
	return &conn{p1r, p2w}, &conn{p2r, p1w}
}

type conn struct {
	io.ReadCloser
	io.WriteCloser
}

func (c *conn) Close() error {
	return multierr.Combine(
		c.ReadCloser.Close(),
		c.WriteCloser.Close())
}
