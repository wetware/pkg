package wasm

import (
	"context"
	"io"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/ww/internal/api/iostream"
	"github.com/wetware/ww/internal/api/wasm"
	"github.com/wetware/ww/pkg/iostream"
)

// Stdin specifies the process's standard input.
//
// If Stdin is nil, the process reads from the null device (os.DevNull).
//
// If Stdin is an *os.File, the process's standard input is connected
// directly to that file.
//
// Otherwise, during the execution of the command a separate
// goroutine reads from Stdin and delivers that data to the command
// over a pipe. In this case, Wait does not complete until the goroutine
// stops copying, either because it has reached the end of Stdin
// (EOF or a read error) or because writing to the pipe returned an error.
func (c RunContext) WithStdin(r io.Reader) RunContext {
	return c.Bind(Stdin(iostream.NewProvider(r)))
}

// Stdout and Stderr specify the process's standard output and error.
//
// If either is nil, Run connects the corresponding file descriptor
// to the null device (os.DevNull).
//
// If either is an *os.File, the corresponding output from the process
// is connected directly to that file.
//
// Otherwise, during the execution of the command a separate goroutine
// reads from the process over a pipe and delivers that data to the
// corresponding Writer. In this case, Wait does not complete until the
// goroutine reaches EOF or encounters an error.
//
// If Stdout and Stderr are the same writer, and have a type that can
// be compared with ==, at most one goroutine at a time will call Write.
func (c RunContext) WithStdout(w io.Writer) RunContext {
	return c.Bind(Stdout(iostream.New(w)))
}

// Stderr behaves analogously to Stdout.
func (c RunContext) WithStderr(w io.Writer) RunContext {
	return c.Bind(Stderr(iostream.New(w)))
}

// Stdin behaves analogously to Stdin, except that it takes
// an iostream.Provider.
func Stdin(p iostream.Provider) Param {
	return func(rc wasm.Runtime_Context) error {
		return rc.SetStdin(api.Provider(p))
	}
}

// Stdout behaves analogously to Stdout, except that it takes
// an iostream.Stream.
func Stdout(s iostream.Stream) Param {
	return func(rc wasm.Runtime_Context) error {
		return rc.SetStdout(api.Stream(s))
	}
}

// Stderr behaves analogously to Stderr, except that it takes
// an iostream.Stream.
func Stderr(s iostream.Stream) Param {
	return func(rc wasm.Runtime_Context) error {
		return rc.SetStderr(api.Stream(s))
	}
}

func stdin(c wasm.Runtime_Context) io.Reader {
	r := c.Stdin().AddRef()
	return input(iostream.Provider(r))
}

func stdout(c wasm.Runtime_Context) io.Writer {
	w := c.Stdout().AddRef()
	return output(iostream.Stream(w))
}

func stderr(c wasm.Runtime_Context) io.Writer {
	w := c.Stderr().AddRef()
	return output(iostream.Stream(w))
}

func input(p iostream.Provider) io.Reader {
	if invalid(p) {
		return nil
	}

	pr, pw := io.Pipe()
	go func() {
		f, release := p.Provide(context.TODO(), iostream.New(pw))
		defer release()

		if err := f.Await(context.TODO()); err != nil {
			pw.CloseWithError(err)
		}
	}()

	return pr
}

func output(s iostream.Stream) io.Writer {
	if invalid(s) {
		return nil
	}

	return s.Writer(context.TODO())
}

func invalid[T ~capnp.ClientKind](t T) bool {
	return !capnp.Client(t).IsValid()
}
