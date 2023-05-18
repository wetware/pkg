package system

import (
	"context"
	"errors"
	"net"
	"syscall"

	"github.com/lthibault/log"
	"github.com/stealthrocket/wazergo"
	. "github.com/stealthrocket/wazergo/types"
)

// Declare the host module from a set of exported functions.
var HostModule wazergo.HostModule[*Module] = functions{
	"_info":  wazergo.F1((*Module).info),
	"_close": wazergo.F0((*Module).close),
	"_recv":  wazergo.F1((*Module).recv),
	"_send":  wazergo.F1((*Module).send),
}

// The `functions` type impements `HostModule[*Module]`, providing the
// module name, map of exported functions, and the ability to create instances
// of the module type.
type functions wazergo.Functions[*Module]

func (f functions) Name() string {
	return "ww"
}

func (f functions) Functions() wazergo.Functions[*Module] {
	return (wazergo.Functions[*Module])(f)
}

func (f functions) Instantiate(ctx context.Context, opts ...Option) (*Module, error) {
	mod := &Module{}
	wazergo.Configure(mod, opts...)
	return mod, nil
}

type Option = wazergo.Option[*Module]

func WithLogger(log log.Logger) Option {
	return wazergo.OptionFunc(func(m *Module) {
		m.log = log
	})
}

func WithPipe(pipe net.Conn) Option {
	return wazergo.OptionFunc(func(m *Module) {
		m.conn = pipe
	})
}

// Module will be the Go type we use to maintain the state of our module
// instances.
type Module struct {
	log  log.Logger
	conn net.Conn // guest-end of the pipe
}

func (m *Module) Close(context.Context) error {
	return m.conn.Close()
}

func (m *Module) info(ctx context.Context, s String) None {
	m.log.Info(string(s))
	return None{}
}

func (m *Module) close(context.Context) Error {
	if err := m.conn.Close(); err != nil {
		return Fail(err)
	}

	return OK
}

func (m *Module) recv(ctx context.Context, b Bytes) (res Uint64) {
	return ioResult(m.conn.Read(b))
}

func (m *Module) send(ctx context.Context, b Bytes) (res Uint64) {
	return ioResult(m.conn.Write(b))
}

// borrowed from wazergo.
// TODO:  maybe submit a PR that makes this public?  Good beginner's task.
func errno(err error) Errno {
	if err == nil {
		return 0
	}
	for {
		switch e := errors.Unwrap(err).(type) {
		case nil:
			return -1 // unknown, just don't return 0
		case interface{ Errno() int32 }:
			return Errno(e.Errno())
		case syscall.Errno:
			return Errno(e)
		default:
			err = e
		}
	}
}

func ioResult(n int, err error) (u64 Uint64) {
	u64 = Uint64(n << 32)           // use left-most 32 bits to encode n
	return u64 | Uint64(errno(err)) // use right-most 32 bits errno
}
