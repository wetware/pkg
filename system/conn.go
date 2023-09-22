package system

import (
	"io"
	"net"
	"sync"

	"go.uber.org/multierr"
)

var (
	ErrOverflow  net.Error = overflowError{}
	ErrUnderflow net.Error = underflowError{}
)

type pipe struct {
	mu sync.Mutex

	// buf contains the data in the pipe.  It is a ring buffer of fixed capacity,
	// with r and w pointing to the offset to read and write, respsectively.
	//
	// Data is read between [r, w) and written to [w, r), wrapping around the end
	// of the slice if necessary.
	//
	// The buffer is empty if r == len(buf), otherwise if r == w, it is full.
	//
	// w and r are always in the range [0, cap(buf)) and [0, len(buf)].
	buf  []byte
	w, r int

	closed      bool
	writeClosed bool
}

func Pipe() (io.ReadWriteCloser, io.ReadWriteCloser) {
	p1, p2 := newPipe(1024), newPipe(1024)
	return &conn{p1, p2}, &conn{p2, p1}
}

func newPipe(sz int) *pipe {
	return &pipe{buf: make([]byte, 0, sz)}
}

func (p *pipe) empty() bool {
	return p.r == len(p.buf)
}

// func (p *pipe) full() bool {
// 	return p.r < len(p.buf) && p.r == p.w
// }

func (p *pipe) Read(b []byte) (n int, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch {
	case p.closed:
		return 0, io.ErrClosedPipe

	case p.writeClosed:
		return 0, io.EOF

	case p.empty():
		return 0, ErrUnderflow

	default:
		n = copy(b, p.buf[p.r:len(p.buf)])
		p.r += n
		if p.r == cap(p.buf) {
			p.r = 0
			p.buf = p.buf[:p.w]
		}

		return
	}
}

func (p *pipe) Write(b []byte) (n int, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch {
	case p.closed || p.writeClosed:
		return 0, io.ErrClosedPipe

	default:
		end := cap(p.buf)
		if p.w < p.r {
			end = p.r
		}
		n = copy(p.buf[p.w:end], b)

		// overflow?
		if n < len(b) {
			return n, ErrOverflow
		}

		if p.w += n; p.w > len(p.buf) {
			p.buf = p.buf[:p.w]
		}

		if p.w == cap(p.buf) {
			p.w = 0
		}
	}

	return
}

func (p *pipe) Close() error {
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()
	return nil
}

func (p *pipe) closeWrite() error {
	p.mu.Lock()
	p.writeClosed = true
	p.mu.Unlock()
	return nil
}

type conn struct {
	io.Reader
	io.Writer
}

func (c *conn) Close() error {
	return multierr.Combine(
		c.Reader.(*pipe).Close(),
		c.Writer.(*pipe).closeWrite())
}

type overflowError struct{}

func (overflowError) Timeout() bool   { return true }
func (overflowError) Temporary() bool { return false }
func (overflowError) Error() string   { return "overflow" }
func (overflowError) Errno() int32    { return 1 }

type underflowError struct{}

func (underflowError) Timeout() bool   { return true }
func (underflowError) Temporary() bool { return false }
func (underflowError) Error() string   { return "underflow" }
func (underflowError) Errno() int32    { return 2 }
