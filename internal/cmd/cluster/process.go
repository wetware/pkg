package cluster

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli/v2"
	process "github.com/wetware/ww/pkg/process/client"
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

		var stdin io.Reader = os.Stdin
		var stdout, stderr io.Writer

		ctx := c.Context
		// Load the name of the entry function and the WASM file containing the module to run
		entryFunction := c.String(_func)
		binary, err := os.ReadFile(c.String(_module))
		if err != nil {
			return err
		}

		// Obtain an executor and spawn a process
		executor, release := node.Executor(ctx)
		defer release()
		proc := process.MakeProcess(ctx, logger, executor, binary, entryFunction)
		defer proc.Close(ctx)

		// Select the output
		if c.Bool(_json) {
			stdout = new(bytes.Buffer)
			stderr = new(bytes.Buffer)
		} else {
			stdout = os.Stdout
			stderr = os.Stderr
		}

		// Run the process
		errs := proc.Run(ctx, stdin, stdout, stderr)

		// Output the results
		if c.Bool(_json) {
			err = outputToJSON(stdout.(*bytes.Buffer), stderr.(*bytes.Buffer), errs)
		} else {
			err = outputToLog(errs)
		}

		return err
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

func outputToLog(errs []error) error {
	var err error
	if errs != nil && len(errs) > 0 {
		for _, err := range errs {
			logger.Error(err)
		}
		err = runError
	}
	return err
}
