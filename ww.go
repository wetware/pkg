package ww

import (
	"context"
	"errors"
	"io"
	"runtime"

	// "github.com/spy16/slurp"
	// "github.com/spy16/slurp/core"
	// "github.com/spy16/slurp/reader"
	// "github.com/spy16/slurp/repl"

	"github.com/lthibault/log"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	casm "github.com/wetware/casm/pkg"

	// "github.com/wetware/ww/api"
	"github.com/wetware/ww/pkg/anchor"
	"github.com/wetware/ww/system"
	"go.uber.org/fx"
)

const Version = "0.1.0"

type Ww struct {
	fx.In `ignore-unexported:"true"`

	Log    log.Logger
	Name   string
	Stdin  io.Reader `name:"stdin"`
	Stdout io.Writer `name:"stdout"`
	Stderr io.Writer `name:"stderr"`
	ROM    system.ROM
	Vat    casm.Vat
	Root   *anchor.Node
}

func (ww Ww) String() string {
	return ww.Name
}

func (ww Ww) Exec(ctx context.Context) error {
	r := wazero.NewRuntimeWithConfig(ctx, wazero.
		NewRuntimeConfigCompiler().
		WithCloseOnContextDone(true))
	c, err := wasi_snapshot_preview1.Instantiate(ctx, r)
	if err != nil {
		return err
	}
	defer c.Close(ctx)

	compiled, err := r.CompileModule(ctx, ww.ROM)
	if err != nil {
		return err
	}
	defer compiled.Close(ctx)

	mod, err := r.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().
		WithOsyield(runtime.Gosched).
		WithStartFunctions(). // don't call _start until later
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime().
		WithName(ww.Name).
		WithEnv("ns", ww.Name).
		WithStdin(ww.Stdin).
		WithStdout(ww.Stdout).
		WithStderr(ww.Stderr).
		WithFSConfig(wazero.
			NewFSConfig().
			WithFSMount(ww.FS(ctx), ww.Name))) // mount ww to ./ww
	if err != nil {
		return err
	}
	defer mod.Close(ctx)

	if fn := mod.ExportedFunction("_start"); fn != nil {
		_, err = fn.Call(ctx)
		return err
	}

	return errors.New("missing export: _start")
}

func (ww Ww) FS(ctx context.Context) system.FS {
	return system.FS{
		Ctx:  ctx,
		Log:  ww.Log,
		Root: ww.Root,
	}
}
