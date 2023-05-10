package cluster

import (
	"io"
	"os"

	"github.com/urfave/cli/v2"
)

func run() *cli.Command {
	return &cli.Command{
		Name:      "run",
		Usage:     "compile and run a WASM module",
		ArgsUsage: "<path> (defaults to stdin)",
		Before:    setup(),
		After:     teardown(),
		Action:    runAction(),
	}
}

func runAction() cli.ActionFunc {
	return func(c *cli.Context) error {
		ctx := c.Context

		// Load the name of the entry function and the WASM file containing the module to run
		src, err := bytecode(c)
		if err != nil {
			return err
		}

		// Obtain an executor and spawn a process
		executor, release := node.Executor(ctx)
		defer release()

		proc, release := executor.Exec(ctx, src)
		defer release()

		return proc.Wait(ctx)
	}
}

func bytecode(c *cli.Context) ([]byte, error) {
	if c.Args().Len() > 0 {
		return os.ReadFile(c.Args().First()) // file path
	}

	return io.ReadAll(c.App.Reader) // stdin
}
