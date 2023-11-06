package system

import (
	"bytes"
	"context"
	"io"
	"sync"

	"capnproto.org/go/capnp/v3/exp/bufferpool"
	"github.com/stealthrocket/wazergo"
	"github.com/stealthrocket/wazergo/types"
	"github.com/tetratelabs/wazero"
)

// Declare the host module from a set of exported functions.
var HostModule wazergo.HostModule[*Host] = functions{
	"sock_close": wazergo.F0((*Host).SockClose),
	"sock_send":  wazergo.F1((*Host).SockSend),
}

func Instantiate(ctx context.Context, r wazero.Runtime, vat io.WriteCloser) (*wazergo.ModuleInstance[*Host], error) {
	return wazergo.Instantiate(ctx, r, HostModule,
		WithWriter(vat))
}

// The `functions` type impements `HostModule[*Module]`, providing the
// module name, map of exported functions, and the ability to create instances
// of the module type.
type functions wazergo.Functions[*Host]

func (f functions) Name() string {
	return "ww"
}

func (f functions) Functions() wazergo.Functions[*Host] {
	return (wazergo.Functions[*Host])(f)
}

func (f functions) Instantiate(ctx context.Context, opts ...Option) (*Host, error) {
	mod := &Host{}
	wazergo.Configure(mod, opts...)
	return mod, nil
}

type Option = wazergo.Option[*Host]

func WithWriter(w io.WriteCloser) Option {
	return wazergo.OptionFunc(func(h *Host) {
		h.Writer = w
	})
}

// Host will be the Go type we use to maintain the state of our module
// instances.
type Host struct {
	Writer io.WriteCloser
}

func (m *Host) Close(context.Context) error {
	return m.Writer.Close()
}

func (m *Host) SockClose(context.Context) types.Error {
	if err := m.Writer.Close(); err != nil {
		return types.Fail(err)
	}

	return types.OK
}

func (m *Host) SockSend(_ context.Context, b types.Bytes) types.Error {
	if err := writeTo(m.Writer, b); err != nil {
		return types.Fail(err)
	}

	return types.OK
}

func writeTo(w io.Writer, b []byte) error {
	rd := readerFor(b)
	defer rd.Release()

	buf := bufferpool.Default.Get(1024)
	defer bufferpool.Default.Put(buf)

	_, err := io.CopyBuffer(w, bytes.NewReader(b), buf)
	return err
}

type reader struct{ bytes.Reader }

func readerFor(b []byte) *reader {
	rd := pool.Get().(*reader)
	rd.Reset(b)
	return rd
}

func (r *reader) Release() {
	r.Reader.Reset(nil)
	pool.Put(r)
}

var pool = sync.Pool{
	New: func() any {
		return new(reader)
	},
}

// type Module struct {
// 	api.Module
// 	Transport io.Closer
// }

// func (m Module) Close(ctx context.Context) error {
// 	return multierr.Combine(
// 		m.Module.Close(ctx),
// 		m.Transport.Close())
// }

// func Instantiate(ctx context.Context, r wazero.Runtime, vat io.WriteCloser) (*Module, error) {
// 	mod, err := bind(ctx, r.NewHostModuleBuilder("ww"), vat)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &Module{
// 		Module:    mod,
// 		Transport: vat,
// 	}, nil
// }

// func bind(ctx context.Context, b wazero.HostModuleBuilder, vat Vat) (api.Module, error) {
// 	return module(b,
// 		&sockCloser{vat},
// 		&sockWriter{vat},
// 	).Instantiate(ctx)
// }

// func module(b wazero.HostModuleBuilder, exports ...export) wazero.HostModuleBuilder {
// 	for _, e := range exports {
// 		b = b.NewFunctionBuilder().
// 			WithGoModuleFunction(e, e.Params(), e.Results()).
// 			WithParameterNames(e.ParamNames()...).
// 			WithResultNames(e.ResultNames()...).
// 			WithName(e.String()).
// 			Export(e.String())
// 	}
// 	return b
// }

// type export interface {
// 	api.GoModuleFunction
// 	String() string
// 	Params() []api.ValueType
// 	Results() []api.ValueType
// 	ParamNames() []string
// 	ResultNames() []string
// }

// type sockCloser struct{ io.Closer }

// func (sockCloser) String() string {
// 	return "sock_close"
// }

// func (sockCloser) Params() []api.ValueType  { return nil }
// func (sockCloser) ParamNames() []string     { return nil }
// func (sockCloser) Results() []api.ValueType { return nil }
// func (sockCloser) ResultNames() []string    { return nil }

// func (c sockCloser) Call(ctx context.Context, mod api.Module, stack []uint64) {
// 	if err := c.Close(); err != nil {
// 		slog.ErrorContext(ctx, err.Error())
// 	}
// }

// type sockWriter struct{ io.Writer }

// func (sockWriter) String() string {
// 	return "sock_write"
// }

// func (sockWriter) Params() []api.ValueType {
// 	return []api.ValueType{
// 		api.ValueTypeI64}
// }

// func (sockWriter) ParamNames() []string {
// 	return []string{
// 		"segment"} // [u32][u32]
// }

// func (sockWriter) Results() []api.ValueType { return nil }
// func (sockWriter) ResultNames() []string    { return nil }

// func (w sockWriter) Call(ctx context.Context, mod api.Module, stack []uint64) {
// 	offset := api.DecodeU32(stack[0])
// 	length := api.DecodeU32(stack[0] >> 32)

// 	w.Send(ctx, mod.Memory(), offset, length)
// }

// func (w sockWriter) Send(ctx context.Context, mem api.Memory, offset, length uint32) {
// 	b, ok := mem.Read(offset, length)
// 	if !ok {
// 		slog.ErrorContext(ctx, "segment out-of-bounds",
// 			"off", offset,
// 			"len", length)
// 		return
// 	}

// 	n, err := io.Copy(w, bytes.NewReader(b))
// 	if err != nil {
// 		slog.ErrorContext(ctx, "failed to send message to host",
// 			"reason", err,
// 			"n", n)
// 		return
// 	}

// 	slog.DebugContext(ctx, "delivered message to host",
// 		"size", n)
// }
