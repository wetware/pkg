package unix

import (
	"fmt"
	"io"
	"path/filepath"

	"capnproto.org/go/capnp/v3"

	iostream_api "github.com/wetware/ww/internal/api/iostream"
	"github.com/wetware/ww/internal/api/proc"
	"github.com/wetware/ww/pkg/iostream"
)

type CommandFunc func(proc.Executor_exec_Params) error

func New() CommandFunc {
	return func(ps proc.Executor_exec_Params) error {
		arena := ps.Message().Arena
		_, s, err := capnp.NewMessage(arena)
		if err != nil {
			return err
		}

		spec, err := proc.NewRootUnix_Command(s)
		if err != nil {
			return err
		}

		return ps.SetParam(spec.ToPtr())
	}
}

// Command is a convenience function for building simple Unix commands
// comprising a command path and a list of arguments.
//
// Additional parameters MAY be chained using Bind, as usual.
func Command(command string, args ...string) CommandFunc {
	return New().
		Bind(Path(command)).
		Bind(Args(args...))
}

// Bind a parameter to the unix command.
func (cmd CommandFunc) Bind(f func(proc.Unix_Command) error) CommandFunc {
	return func(ps proc.Executor_exec_Params) error {
		if err := cmd(ps); err != nil {
			return err
		}

		p, err := ps.Param()
		if err != nil {
			return err
		}

		return f(proc.Unix_Command(p.Struct()))
	}
}

// Path is the path of the command to run.
//
// This is the only field that must be set to a non-zero
// value. If Path is relative, it is evaluated relative
// to Dir.
func Path(path string) func(proc.Unix_Command) error {
	path = filepath.Clean(path)

	return func(c proc.Unix_Command) error {
		err := c.SetPath(path)
		return maybe("path:", err)
	}
}

// Dir specifies the working directory of the command.
// If Dir is the empty string, Run runs the command in the
// calling process's current directory.
func Dir(dir string) func(proc.Unix_Command) error {
	dir = filepath.Clean(dir)

	return func(c proc.Unix_Command) error {
		err := c.SetDir(dir)
		return maybe("dir:", err)
	}
}

// Args holds command line arguments, including the command as Args[0].
// If the Args field is empty or nil, Run uses {Path}.
//
// In typical use, both Path and Args are set by calling Command.
func Args(args ...string) func(proc.Unix_Command) error {
	return func(c proc.Unix_Command) error {
		as, err := c.NewArgs(int32(len(args)))
		if err != nil {
			return maybe("args:", err)
		}

		for i, arg := range args {
			if err = as.Set(i, arg); err != nil {
				break
			}
		}

		return maybe("args:", err)
	}
}

// Env specifies the environment of the process.
// Each entry is of the form "key=value".
// If Env is nil, the new process uses the current process's
// environment.
// If Env contains duplicate environment keys, only the last
// value in the slice for each duplicate key is used.
// As a special case on Windows, SYSTEMROOT is always added if
// missing and not explicitly set to the empty string.
func Env(env ...string) func(proc.Unix_Command) error {
	size := int32(len(env))

	return func(c proc.Unix_Command) error {
		e, err := c.NewEnv(size)
		if err != nil {
			return maybe("env:", err)
		}

		for i, s := range env {
			if err = e.Set(i, s); err != nil {
				break
			}
		}

		return maybe("env:", err)
	}
}

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
func Stdin(r io.Reader) func(proc.Unix_Command) error {
	return StdinCap(iostream.NewProvider(r))
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
func Stdout(w io.Writer) func(proc.Unix_Command) error {
	return StdoutCap(iostream.New(w))
}

// Stderr behaves analogously to Stdout.
func Stderr(w io.Writer) func(proc.Unix_Command) error {
	return StderrCap(iostream.New(w))
}

// StdinCap behaves analogously to Stdin, except that it takes
// an iostream.Provider.
func StdinCap(p iostream.Provider) func(proc.Unix_Command) error {
	return func(c proc.Unix_Command) error {
		return c.SetStdin(iostream_api.Provider(p))
	}
}

// StdoutCap behaves analogously to Stdout, except that it takes
// an iostream.Stream.
func StdoutCap(s iostream.Stream) func(proc.Unix_Command) error {
	return func(c proc.Unix_Command) error {
		return c.SetStdout(iostream_api.Stream(s))
	}
}

// StderrCap behaves analogously to Stderr, except that it takes
// an iostream.Stream.
func StderrCap(s iostream.Stream) func(proc.Unix_Command) error {
	return func(c proc.Unix_Command) error {
		return c.SetStderr(iostream_api.Stream(s))
	}
}

func maybe(prefix string, err error) error {
	if err != nil {
		err = fmt.Errorf("%s %w", prefix, err)
	}

	return err
}
