package cluster

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/wetware/ww/pkg/process"
)

const (
	_module = "module"
	_func   = "function"
)

var runError = errors.New("Run failed.")

func run() *cli.Command {
	return &cli.Command{
		Name:  "run",
		Usage: "run a WASM module",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     _module,
				Aliases:  []string{"m"},
				Usage:    "path to the file containing compiled WASM module",
				Required: true,
			},
			&cli.StringFlag{
				Name:     _func,
				Aliases:  []string{"f"},
				Usage:    "name of the function to run within the WASM module",
				Required: true,
			},
			&boolFlag,
		},
		Before: setup(),
		After:  teardown(),
		Action: runAction(),
	}
}

func runAction() cli.ActionFunc {
	return func(c *cli.Context) error {
		ctx := c.Context
		// Load the name of the entry function and the WASM file containing the module to run
		binary, err := os.ReadFile(c.String(_module))
		if err != nil {
			return err
		}

		// Obtain an executor and spawn a process
		executor, release := node.Executor(ctx)
		defer release()

		proc, release := executor.Spawn(ctx, process.Config{
			Executable: binary,
			EntryPoint: c.String(_func),
		})
		defer release()
		defer proc.Close(ctx)

		if err := proc.Start(ctx); err != nil {
			return err
		}
		defer proc.Stop(ctx)

		return proc.Wait(ctx)
	}
}

type results struct {
	Stdout string   `json:"stdout"`
	Stderr string   `json:"stderr"`
	Errs   []string `json:"errors"`
}

func outputToJSON(output *bytes.Buffer, errorOutput *bytes.Buffer, errs []error) error {
	var err error
	errStrings := make([]string, len(errs))
	for i, e := range errs {
		errStrings[i] = e.Error()
	}
	results := results{
		Stdout: output.String(),
		Stderr: errorOutput.String(),
		Errs:   errStrings,
	}
	content, err := json.Marshal(results)
	if err != nil {
		return err
	}
	fmt.Println(string(content))
	return nil
}
