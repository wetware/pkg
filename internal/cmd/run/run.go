package run

import (
	"crypto/rand"
	"errors"
	"io"
	"net"
	"os"

	"capnproto.org/go/capnp/v3"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/urfave/cli/v2"
	"github.com/wetware/ww/pkg/csp/fs"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:   "run",
		Usage:  "execute a local webassembly process",
		Before: setup(),
		After:  teardown(),
		Action: run(),
	}
}

var (
	r wazero.Runtime
)

func setup() cli.BeforeFunc {
	return func(c *cli.Context) error {
		r = wazero.NewRuntime(c.Context)
		wasi_snapshot_preview1.MustInstantiate(c.Context, r)
		return nil
	}
}

func teardown() cli.AfterFunc {
	return func(c *cli.Context) error {
		return r.Close(c.Context)
	}
}

func run() cli.ActionFunc {
	return func(c *cli.Context) error {
		b, err := bytecode(c)
		if err != nil {
			return err
		}

		host, guest := net.Pipe()

		module, err := r.InstantiateWithConfig(c.Context, b, wazero.NewModuleConfig().
			WithRandSource(rand.Reader).
			WithStartFunctions(). // disable auto-calling of _start
			WithStdout(c.App.Writer).
			WithStderr(c.App.ErrWriter).
			WithFS(fs.FS{Host: host, Guest: guest, BootstrapClient: capnp.Client{}}))
		if err != nil {
			return err
		}

		fn := module.ExportedFunction("_start")
		if fn == nil {
			return errors.New("ww: missing export: _start")
		}

		_, err = fn.Call(c.Context)
		return err
	}
}

func bytecode(c *cli.Context) ([]byte, error) {
	if c.Args().Len() > 0 {
		return os.ReadFile(c.Args().First()) // file path
	}

	return io.ReadAll(c.App.Reader) // stdin
}
