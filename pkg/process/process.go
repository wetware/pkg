package process

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"

	capnp "capnproto.org/go/capnp/v3"
	"github.com/lthibault/log"
	wazero_api "github.com/tetratelabs/wazero/api"

	iostream_api "github.com/wetware/ww/internal/api/iostream"
	api "github.com/wetware/ww/internal/api/process"
	proc_errors "github.com/wetware/ww/pkg/process/errors"
)

// Process represents the execution of a function in a WASM module.
type Process struct {
	function wazero_api.Function // entry function
	id       string              // process id
	io       processIo
	logger   log.Logger

	exitWaiters  []chan struct{}     // list of channels waiting for process exit
	releaseFuncs []capnp.ReleaseFunc // pending releases
	runCancel    context.CancelFunc  // cancellation call for runContext
	runContext   context.Context     // context for the process runtime
	runDone      chan error          // channel containing result of run
}

// addExitWaiter returns a channel that will produce a value when
// the proces exits.
func (p *Process) addExitWaiter() chan struct{} {
	exit := make(chan struct{}, 1)
	p.exitWaiters = append(p.exitWaiters, exit)
	return exit
}

// addRelease adds a release function to the pending releases.
func (p *Process) addRelease(release capnp.ReleaseFunc) {
	p.releaseFuncs = append(p.releaseFuncs, release)
}

// Close should always be called after the process is done.
func (p *Process) Close(ctx context.Context, call api.Process_close) error {
	return p.close(ctx)
}

// close performs any missing cleanup operation including potentially
// cancelling a running process, notifying all exitWaiters and calling
// pending release functions.
func (p *Process) close(ctx context.Context) error {
	defer p.release()
	defer p.exit(ctx)
	defer p.runCancel()
	return nil
}

// exit will notify all exit waiters.
func (p *Process) exit(ctx context.Context) {
	for _, exit := range p.exitWaiters {
		select {
		case exit <- struct{}{}:
			continue
		case <-ctx.Done():
			return
		}
	}
}

// Id of the process.
func (p *Process) Id() string {
	return p.id
}

// Input returns the input stream of the process.
func (p *Process) Input(ctx context.Context, call api.Process_input) error {
	results, err := call.AllocResults()
	if err != nil {
		return err
	}
	err = results.SetStream(iostream_api.Stream(p.io.in))
	return err
}

// Output provides an stream with the process output and returns the
// contents of the process stderr after it finishes.
func (p *Process) Output(ctx context.Context, call api.Process_output) error {
	var err error
	results, err := call.AllocResults()
	if err != nil {
		return err
	}

	call.Go()
	stream := call.Args().Stream()
	f, release := p.io.out.Provide(ctx, func(p iostream_api.Provider_provide_Params) error {
		return p.SetStream(stream)
	})
	defer release()

	exit := p.addExitWaiter()
	select {
	case <-ctx.Done():
		err = ctx.Err()
		break
	case <-exit:
	case <-f.Done():
		break
	}

	if err != nil || p.io.errBuffer.Len() > 0 {
		results.SetError(p.io.errBuffer.String())
	} else {
		results.SetError(proc_errors.Nil.Error())
	}

	return err
}

// run the process and write to p.runDone after it is done.
func (p *Process) run(ctx context.Context) {
	var err error

	// send the signal after the process finishes running
	defer func() {
		select {
		case p.runDone <- err:
			break
		case <-ctx.Done():
			p.runCancel()
		}
	}()

	// close input and output pipes
	defer func() {
		// data can sometimes be lost if outW.Close() is called too early
		// TODO mikel find cause
		p.io.outW.Close()
	}()
	defer p.io.in.Close(p.runContext, nil)

	_, err = p.function.Call(p.runContext)
}

// release calls all pending release functions.
func (p *Process) release() {
	for _, releaseFunc := range p.releaseFuncs {
		defer releaseFunc()
	}
}

// Stop calls the runtime cancellation function.
func (p *Process) Stop(ctx context.Context, call api.Process_stop) error {
	p.runCancel()
	return nil
}

// Start the process in the background.
func (p *Process) Start(ctx context.Context, call api.Process_start) error {
	go p.run(ctx)
	return nil
}

// Wait for the process to finish running.
func (p *Process) Wait(ctx context.Context, call api.Process_wait) error {
	results, err := call.AllocResults()
	if err != nil {
		err = p.wait(ctx)
		if err == nil {
			results.SetError(proc_errors.Nil.Error())
		} else {
			results.SetError(err.Error())
		}
	}
	return err
}

// wait for the process to finish running.
func (p *Process) wait(ctx context.Context) error {
	var err error
	select {
	case err = <-p.runDone:
		break
	case <-ctx.Done():
		break
	}
	return err
}

// moduleId retuns a shortened md5hash of the module
func moduleId(binary []byte) string {
	hash := md5.Sum(binary)
	return hex.EncodeToString(hash[:])[:6]
}

// processId returns a unique ID for a module
func processId(moduleId string, funcName string) string {
	return fmt.Sprintf("%s:%s", moduleId, funcName)
}

// randomId produces a 6 character random string
func randomId() string {
	rand.Seed(time.Now().Unix())
	charset := "abcdefghijklmnopqrstuvwxyz"
	length := 6

	id := make([]byte, length)
	for i := 0; i < length; i++ {
		id[i] = charset[rand.Intn(len(charset))]
	}

	return string(id)
}
