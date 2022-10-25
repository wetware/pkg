package run

import (
	"fmt"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/urfave/cli/v2"
	"github.com/wetware/ww/pkg/csp/wasm"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:   "run",
		Usage:  "compile and run a process",
		Action: run(),
	}
}

func run() cli.ActionFunc {
	return func(c *cli.Context) error {
		r, err := runtime(c)
		if err != nil {
			return fmt.Errorf("runtime: %w", err)
		}

		b, err := os.ReadFile(c.Args().First())
		if err != nil {
			return fmt.Errorf("fopen: %w", err)
		}

		p, release := r.Exec(c.Context, wasm.NewContext(b).
			WithStdin(c.App.Reader).
			WithStdout(c.App.Writer).
			WithStderr(c.App.ErrWriter))
		defer release()

		f, release := p.Run(c.Context)
		defer release()

		return f.Err()
	}
}

func runtime(c *cli.Context) (wasm.Runtime, error) {
	// TODO:  check for a -dial flag and connect to cluster

	return wasm.RuntimeFactory{
		Config: config(c),
	}.Runtime(c.Context), nil
}

func config(c *cli.Context) wazero.RuntimeConfig {
	return wazero.NewRuntimeConfigCompiler()
}
