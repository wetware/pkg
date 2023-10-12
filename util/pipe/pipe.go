package pipe

import (
	"context"
	"errors"
	"io"
	"sync"
)

const bufsize = 1 << 13

var ErrInterrupt = errors.New("interrupt")

func New() (host, guest *Pipe) {
	left, right := newBuffer(), newBuffer()

	host = &Pipe{
		Reader: left,
		Writer: right,
		Closer: right,
	}

	guest = &Pipe{
		Reader: right,
		Writer: left,
		Closer: left,
	}

	return
}

type Pipe struct {
	io.Reader
	io.Writer
	io.Closer
}

func (p Pipe) FlushReader(ctx context.Context) error {
	select {
	case <-p.Reader.(*buffer).ReadReady():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p Pipe) FlushWriter(ctx context.Context) error {
	select {
	case <-p.Writer.(*buffer).WriteReady():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type buffer struct {
	// TODO(performance):  I'm pretty sure it's safe to remove the mutex.
	mu                    sync.Mutex
	r, w                  uint32
	readReady, writeReady chan struct{}
	buf                   [bufsize]byte // must be power of 2
	closed                bool
}

func newBuffer() *buffer {
	buf := &buffer{
		readReady:  make(chan struct{}, 1),
		writeReady: make(chan struct{}, 1),
	}
	buf.writeReady <- struct{}{}

	return buf
}

func (b *buffer) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.closed = true
	return nil
}

func (b *buffer) ReadReady() <-chan struct{} {
	return b.readReady
}

func (b *buffer) Read(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return 0, io.EOF
	}

	wasFull := b.full()

	for i := range p {
		if b.empty() {
			err = ErrInterrupt // underflow
			break
		}

		n++
		b.r++
		p[i] = b.buf[b.mask(b.r)] // copy
	}

	if wasFull && n > 0 {
		select {
		case b.writeReady <- struct{}{}:
		default:
		}
	}

	return
}

func (b *buffer) WriteReady() <-chan struct{} {
	return b.writeReady
}

func (b *buffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return 0, errors.New("closed")
	}

	wasEmpty := b.empty()

	for _, x := range p {
		if b.full() {
			err = ErrInterrupt // overflow
			break
		}

		n++
		b.w++
		b.buf[b.mask(b.w)] = x
	}

	if wasEmpty && n > 0 {
		select {
		case b.readReady <- struct{}{}:
		default:
		}
	}

	return
}

func (b *buffer) mask(u uint32) uint32 {
	return u & (bufsize - 1)
}

func (b *buffer) empty() bool {
	return b.r == b.w
}

func (b *buffer) full() bool {
	return b.size() == bufsize
}

func (b *buffer) size() uint32 {
	return b.w - b.r
}
