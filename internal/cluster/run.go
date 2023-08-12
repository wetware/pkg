package cluster

import (
	"context"
	"io"
	"os"
	"time"

	capnp "capnproto.org/go/capnp/v3"
	"github.com/urfave/cli/v2"
)

const killTimeout = 30 * time.Second

func run(log Logger) *cli.Command {
	return &cli.Command{
		Name:      "run",
		Usage:     "compile and run a WASM module",
		ArgsUsage: "<path> (defaults to stdin)",
		Before:    setup(log),
		After:     teardown(),
		Action:    runAction(log),
	}
}

func runAction(log Logger) cli.ActionFunc {
	return func(c *cli.Context) error {
		ctx := c.Context

		// Load the name of the entry function and the WASM file containing the module to run
		src, err := bytecode(c)
		if err != nil {
			return err
		}

		// Obtain an executor and spawn a process
		executor, release := h.Executor(ctx)
		defer release()

		client := capnp.Client(h.AddRef())
		proc, release := executor.Exec(ctx, src, 0, client)
		defer release()

		waitChan := make(chan error, 1)
		go func() {
			waitChan <- proc.Wait(ctx)
		}()
		select {
		case err = <-waitChan:
			return err
		case <-ctx.Done():
			killChan := make(chan error, 1)
			go func() { killChan <- proc.Kill(context.Background()) }()
			select {
			case err = <-killChan:
				return err
			case <-time.After(killTimeout):
				return err
			}
		}
	}
}

func bytecode(c *cli.Context) ([]byte, error) {
	if c.Args().Len() > 0 {
		return os.ReadFile(c.Args().First()) // file path
	}

	return io.ReadAll(c.App.Reader) // stdin
}