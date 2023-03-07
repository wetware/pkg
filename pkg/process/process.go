package process

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"

	capnp "capnproto.org/go/capnp/v3"
	"github.com/tetratelabs/wazero"
	wasm "github.com/tetratelabs/wazero/api"

	casm "github.com/wetware/casm/pkg"
	api "github.com/wetware/ww/internal/api/process"
)

type Proc api.Process

func (p Proc) AddRef() Proc {
	return Proc(api.Process(p).AddRef())
}

func (p Proc) Release() {
	capnp.Client(p).Release()

}

func (p Proc) Start(ctx context.Context) error {
	f, release := api.Process(p).Start(ctx, nil)
	defer release()

	return casm.Future(f).Await(ctx)
}

func (p Proc) Stop(ctx context.Context) error {
	f, release := api.Process(p).Stop(ctx, nil)
	defer release()

	return casm.Future(f).Await(ctx)
}

func (p Proc) Wait(ctx context.Context) error {
	f, release := api.Process(p).Wait(ctx, nil)
	defer release()

	return p.wait(f)

}

func (p Proc) Close(ctx context.Context) error {
	stop, release := api.Process(p).Stop(ctx, nil)
	defer release()

	wait, release := api.Process(p).Wait(ctx, nil)
	defer release()

	if err := casm.Future(stop).Await(ctx); err != nil {
		return fmt.Errorf("stop: %w", err)
	}

	return p.wait(wait)
}

func (p Proc) wait(f api.Process_wait_Results_Future) error {
	res, err := f.Struct()
	if err != nil {
		return err
	}

	if !res.HasError() {
		return nil
	}

	msg, err := res.Error()
	if err != nil {
		return err
	}

	return Error{Message: msg}
}

// process is the main implementation of the Process capability.
type process struct {
	Runtime   wazero.Runtime
	ByteCode  ByteCode
	EntryFunc string

	cancel context.CancelFunc
	done   chan struct{}
	err    error
}

// Stop calls the runtime cancellation function.
func (p *process) Stop(ctx context.Context, _ api.Process_stop) error {
	if p.cancel == nil {
		return errors.New("not started")
	}

	p.cancel()
	return nil
}

// Start the process in the background.
func (p *process) Start(ctx context.Context, _ api.Process_start) error {
	if p.cancel != nil {
		return errors.New("running")
	}

	name := moduleName(p.ByteCode)
	config := wazero.
		NewModuleConfig().
		WithName(name)

	mod, err := p.loadModule(ctx, name, config)
	if err != nil {
		return err
	}

	entrypoint := mod.ExportedFunction(p.EntryFunc)
	if entrypoint == nil {
		return fmt.Errorf("module %s: %s not found", name, p.EntryFunc)
	}

	p.run(entrypoint)
	return nil
}

func (p *process) loadModule(ctx context.Context, name string, config wazero.ModuleConfig) (wasm.Module, error) {
	if mod := p.Runtime.Module(name); mod != nil {
		return mod, nil
	}

	module, err := p.Runtime.CompileModule(ctx, p.ByteCode)
	if err != nil {
		return nil, err
	}

	return p.Runtime.InstantiateModule(ctx, module, config)
}

// run the process and write to p.runDone after it is done.
func (p *process) run(entrypoint wasm.Function) {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.done = make(chan struct{})
	go func() {
		defer close(p.done)
		defer cancel()

		_, p.err = entrypoint.Call(ctx)
	}()
}

// Wait for the process to finish running.
func (p *process) Wait(ctx context.Context, call api.Process_wait) error {
	results, err := call.AllocResults()
	if err == nil {
		call.Go()
		return p.wait(ctx, results.SetError)
	}

	return err
}

func (p *process) wait(ctx context.Context, setErr func(string) error) error {
	if p.cancel == nil {
		return errors.New("not started")
	}

	select {
	case <-p.done:
	case <-ctx.Done():
		return ctx.Err()
	}

	if p.err != nil {
		return setErr(p.err.Error())
	}

	return nil
}

// moduleName retuns a shortened md5hash of the module
func moduleName(b ByteCode) string {
	prefix := b.String()[:8]

	var buf [4]byte
	rand.Read(buf[:])
	suffix := hex.EncodeToString(buf[:])

	return prefix + "-" + suffix
}
