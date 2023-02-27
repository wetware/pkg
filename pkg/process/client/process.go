package client

import (
	"context"
	"errors"
	"io"

	capnp "capnproto.org/go/capnp/v3"
	"github.com/lthibault/log"

	iostream_api "github.com/wetware/ww/internal/api/iostream"
	api "github.com/wetware/ww/internal/api/process"
	"github.com/wetware/ww/pkg/iostream"
)

// Process is the local implementation of the process used to interact with
// an underlying process capability.
type Process struct {
	capability api.Process // underlying process capability
	logger     log.Logger
	releases   []capnp.ReleaseFunc // pending release functions

	outputDone chan error
	outputErr  chan string
}

// MakeProcess is the default constructor for Process.
func MakeProcess(ctx context.Context, logger log.Logger, executor api.Executor, binary []byte, entryFunction string) *Process {
	future, release := executor.Spawn(ctx, func(e api.Executor_spawn_Params) error {
		var err error

		if err = e.SetBinary(binary); err != nil {
			return err
		}
		return e.SetEntryfunction(entryFunction)
	})

	proc := &Process{
		logger:     logger,
		capability: future.Process(),
		releases:   make([]capnp.ReleaseFunc, 0),

		outputDone: make(chan error, 1),
		outputErr:  make(chan string, 1),
	}

	proc.addRelease(release)

	return proc
}

// addRelease adds a release function to the list of pending releases
// that are called by p.release().
func (p *Process) addRelease(release capnp.ReleaseFunc) {
	p.releases = append(p.releases, release)
}

// bindOutput binds the provided output to the stream that will be provided
// by the server process, as well as notifying either p.outputDone or p.outputErr.
func (p *Process) bindOutput(ctx context.Context, output io.Writer) {
	outputErr, err := p.provideOutput(ctx, output)
	if err != nil {
		select {
		case p.outputDone <- err:
			break
		case <-ctx.Done():
			break
		}
	} else {
		select {
		case p.outputErr <- outputErr:
			break
		case <-ctx.Done():
			break
		}
	}
}

// Cap returns the underlying process capability.
func (p *Process) Cap() api.Process {
	return p.capability
}

// in returns the input stream of the process capability
func (p *Process) in(ctx context.Context) (iostream_api.Stream, error) {
	f, release := p.capability.Input(ctx, nil)
	p.addRelease(release)
	if err := waitForFuncOrCancel(ctx, f.Done); err != nil {
		return iostream_api.Stream{}, err
	}
	return f.Stream(), nil
}

// provideInput to the remote process server.
func (p *Process) provideInput(ctx context.Context, input io.Reader) error {
	inputProvider := iostream_api.Provider(iostream.NewProvider(input))
	inputStream, err := p.in(ctx)
	if err != nil {
		return err
	}
	f, release := inputProvider.Provide(ctx, func(p iostream_api.Provider_provide_Params) error {
		return p.SetStream(inputStream)
	})
	defer release()
	if err = waitForFuncOrCancel(ctx, f.Done); err != nil {
		return err
	}
	return nil
}

// provideOutput calls the remote process.Output providing method,
// waits for it to finish and writes the output to the error buffer.
func (p *Process) provideOutput(ctx context.Context, output io.Writer) (outputErr string, err error) {
	outStream := iostream.New(output)
	defer outStream.Close(ctx) // It should be closed by the provider, but just in case
	f, release := p.capability.Output(ctx, func(params api.Process_output_Params) error {
		return params.SetStream(iostream_api.Stream(outStream))
	})
	p.addRelease(release)

	waitForFuncOrCancel(ctx, f.Done)
	results, err := f.Struct()
	if err != nil {
		return outputErr, err
	}

	outputErr, err = results.Error()
	if err != nil {
		return outputErr, err
	}

	return outputErr, err
}

// Release calls every pending release function.
func (p *Process) release() {
	for _, release := range p.releases {
		defer release()
	}
}

// Run the process with the given input and wait for it to finish.
func (p *Process) Run(ctx context.Context, input io.Reader, output io.Writer) (outputErr string, errs []error) {
	if err := p.Start(ctx, input, output); err != nil {
		return "", []error{err}
	}
	return p.Wait(ctx)
}

// Start the process by binding the remote and local outputs as well as calling
// the Start method of the process capability.
func (p *Process) Start(ctx context.Context, input io.Reader, output io.Writer) error {
	if input == nil || output == nil {
		return errors.New("Process input and output cannot be nil.")
	}
	go p.provideInput(ctx, input)
	go p.bindOutput(ctx, output)

	start, release := p.capability.Start(ctx, nil)
	p.addRelease(release)
	return waitForFuncOrCancel(ctx, start.Done)
}

// waitForOutput waits until all the output from the remote process run is received.
func (p *Process) waitForOutput(ctx context.Context) (outputErr string, err error) {
	select {
	case outputErr = <-p.outputErr:
	case err = <-p.outputDone:
		break
	case <-ctx.Done():
		err = ctx.Err()
		break
	}

	return outputErr, err
}

// waitForProcess waits for the process to finish running.
func (p *Process) waitForProcess(ctx context.Context) error {
	wait, release := p.capability.Wait(ctx, nil)
	p.addRelease(release)
	return waitForFuncOrCancel(ctx, wait.Done)
}

// Wait until the process finishes and all I/O operations are finished.
// Returns the actual error output produced by the process and a slice of wetware errors.
func (p *Process) Wait(ctx context.Context) (outputErr string, errs []error) {
	errs = make([]error, 0)
	if err := p.waitForProcess(ctx); err != nil {
		errs = append(errs, err)
	}
	outputErr, err := p.waitForOutput(ctx)
	if err != nil {
		errs = append(errs, err)
	}
	return outputErr, errs
}

// Close the process by calling all pending releases and closing the underlying
// process capability.
func (p *Process) Close(ctx context.Context) {
	p.release()
	p.capability.Close(ctx, nil)
}

// waitForFuncOrCancel waits until either the channel returned by the function
// produces a value or the context ends. It returns nil in the former case and
// the cause of the context cancelation in the latter.
func waitForFuncOrCancel(ctx context.Context, function func() <-chan struct{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-function():
		return nil
	}
}
