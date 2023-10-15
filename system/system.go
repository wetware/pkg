package system

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math"
	"os"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/sys"
)

type Socket interface {
	io.ReadWriteCloser
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

func Instantiate(ctx context.Context, r wazero.Runtime, sock Socket) (wazero.CompiledModule, error) {
	return bind(ctx, r.NewHostModuleBuilder("ww"), sock)
}

func bind(ctx context.Context, b wazero.HostModuleBuilder, sock Socket) (wazero.CompiledModule, error) {
	return module(b,
		sockRead{sock},
		sockWrite{sock},
		sockClose{sock},
	).Compile(ctx)
}

type export interface {
	api.GoModuleFunction
	String() string
	Params() []api.ValueType
	Results() []api.ValueType
	ParamNames() []string
	ResultNames() []string
}

func module(b wazero.HostModuleBuilder, exports ...export) wazero.HostModuleBuilder {
	for _, e := range exports {
		b = b.NewFunctionBuilder().
			WithGoModuleFunction(e, e.Params(), e.Results()).
			WithParameterNames(e.ParamNames()...).
			WithResultNames(e.ResultNames()...).
			WithName(e.String()).
			Export(e.String())
	}
	return b
}

type sockRead struct{ Socket }

func (sockRead) String() string {
	return "_sock_read"
}

func (sockRead) Params() []api.ValueType {
	return []api.ValueType{
		api.ValueTypeI64, // u64 masked as (u32, u32) pair
		api.ValueTypeI64} // deadline as time.Duration
}

func (sockRead) ParamNames() []string {
	return []string{
		"segment",
		"timeout"}
}

func (sockRead) Results() []api.ValueType {
	return []api.ValueType{
		api.ValueTypeI64} // u64 masked as (n::u32, err:u32) pair
}

func (sockRead) ResultNames() []string {
	return []string{"effect"}
}

func (r sockRead) Call(ctx context.Context, mod api.Module, stack []uint64) {
	seg := segment(stack[0])
	off := seg.Offset()
	size := seg.Size()

	// Set deadline
	d := time.Duration(stack[1])
	if err := r.SetReadDeadline(time.Now().Add(d)); err != nil {
		stack[0] = perform(fail(err), nil)
		return
	}

	// Call read
	if view, ok := mod.Memory().Read(off, size); ok {
		stack[0] = perform(r.Read, view)
	} else {
		slog.Warn("process referenced out-of-bounds segment",
			"method", r.String(),
			"offset", off,
			"size", size)
		stack[0] = perform(fail(errors.New("segment out of bounds")), nil)
	}
}

func perform(effect func([]byte) (int, error), input []byte) uint64 {
	n, err := effect(input)
	return uint64(n)<<32 | status(err)
	// return an 'effect' with the number of bytes set to 'n' and the error
	// set with some appropriate value.
}

func status(err error) uint64 {
	if err == nil {
		return 0
	}

	switch e := err.(type) {
	case *sys.ExitError:
		if errors.Is(e, context.Canceled) {
			return uint64(sys.ExitCodeContextCanceled)
		}
		if errors.Is(e, context.DeadlineExceeded) {
			return uint64(sys.ExitCodeDeadlineExceeded)
		}
		if errors.Is(e, os.ErrDeadlineExceeded) {
			return uint64(sys.ExitCodeDeadlineExceeded)
		}

		return uint64(e.ExitCode())

	case interface{ Errno() uint32 }:
		return uint64(e.Errno())

	default:
		return math.MaxUint32
	}
}

func fail(err error) func(b []byte) (int, error) {
	return func(b []byte) (int, error) {
		return len(b), err
	}
}

type sockWrite struct{ Socket }

func (sockWrite) String() string {
	return "_sock_read"
}

func (sockWrite) Params() []api.ValueType {
	return []api.ValueType{
		api.ValueTypeI64, // u64 masked as (u32, u32) pair
		api.ValueTypeI64} // deadline as time.Duration
}

func (sockWrite) ParamNames() []string {
	return []string{
		"segment",
		"timeout"}
}

func (sockWrite) Results() []api.ValueType {
	return []api.ValueType{
		api.ValueTypeI64} // u64 masked as (n::u32, err:u32) pair
}

func (sockWrite) ResultNames() []string {
	return []string{"effect"}
}

func (w sockWrite) Call(ctx context.Context, mod api.Module, stack []uint64) {
	seg := segment(stack[0])
	off := seg.Offset()
	size := seg.Size()

	// Set deadline
	d := time.Duration(stack[1])
	if err := w.SetWriteDeadline(time.Now().Add(d)); err != nil {
		stack[0] = perform(fail(err), nil)
		return
	}

	// Call write
	if view, ok := mod.Memory().Read(off, size); ok {
		stack[0] = perform(w.Write, view)
	} else {
		slog.Warn("process referenced out-of-bounds segment",
			"method", w.String(),
			"offset", off,
			"size", size)
		stack[0] = perform(fail(errors.New("segment out of bounds")), nil)
	}
}

type sockClose struct{ Socket }

func (sockClose) String() string {
	return "_sock_close"
}

func (sockClose) Params() []api.ValueType {
	return nil
}

func (sockClose) ParamNames() []string {
	return nil
}

func (sockClose) Results() []api.ValueType {
	return nil
}

func (sockClose) ResultNames() []string {
	return nil
}

func (c sockClose) Call(ctx context.Context, _ api.Module, stack []uint64) {
	if err := c.Close(); err != nil {
		slog.Error("failed to close system socket",
			"reason", err)
	}
}

// type errno interface {
// 	Errno() uint32
// }

type segment uint64

func (s segment) Offset() uint32 {
	return uint32(s >> 32)
}

func (s segment) Size() uint32 {
	return uint32(s)
}

type effect uint64

func (s effect) Err() error {
	if eno := s.Errno(); eno != 0 {
		return sys.NewExitError(eno)
	}

	return nil
}

func (s effect) Bytes() uint32 {
	return uint32(s >> 32)
}

func (s effect) Errno() uint32 {
	return uint32(s)
}
