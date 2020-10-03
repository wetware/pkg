// Package proc provides a plugin architecture for Wetware proceses.
package proc

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spy16/parens"
	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/mem"
)

var (
	registry = processRegistry{}

	_ ww.Any = (Proc)(nil)

	_ Proc = (*remoteProc)(nil)
)

// ProcessFactory constructs a process from arguments, and starts it in the
// background.
type ProcessFactory func(env *parens.Env, args []ww.Any) (Proc, error)

// Register a process implementation by mapping a process type to a ProcessFactory.
// Types MUST follow keyword syntax, e.g. ":go" or ":unix".
// If f is nil, the corresponding tag is deleted from the registry.
// Duplicate factories for a given process type will be overwritten.
func Register(procType string, f ProcessFactory) {
	if f == nil {
		delete(registry, procType)
	} else {
		registry[procType] = f
	}
}

// Proc is a generic asynchronous process
type Proc interface {
	ww.Any
	Wait(context.Context) error
}

// Spawn configures a process based on the supplied arguments and then starts it.
func Spawn(env *parens.Env, args ...ww.Any) (Proc, error) {
	if len(args) < 2 {
		return nil, errors.New("expected at least 1 argument, got 0")
	}

	if args[0].MemVal().Type() != api.Value_Which_keyword {
		return nil, errors.New("process-type argument must be of type 'Keyword'")
	}

	procType, err := args[0].MemVal().Raw.Keyword()
	if err != nil {
		return nil, err
	}

	return registry.Spawn(env, procType, args[1:])
}

// FromValue lifts a mem.Value into a first-class Proc type.
// It performs no type-checking, so callers MUST ensure v is a valid Proc.
func FromValue(v mem.Value) Proc { return remoteProc{v} }

type remoteProc struct{ mem.Value }

func (p remoteProc) String() string {
	// TODO(enhancement): provide more info.  Proces ID? Remote host ID?  Current value?
	return fmt.Sprintf("<RemoteProc>")
}

func (p remoteProc) Wait(ctx context.Context) error {
	f, done := p.Raw.Proc().Wait(ctx, func(api.Proc_wait_Params) error { return nil })
	defer done()

	select {
	case <-f.Done():
	case <-ctx.Done():
		return ctx.Err()
	}

	_, err := f.Struct()
	return err
}

type procCap struct{ Proc }

func (p procCap) Wait(ctx context.Context, call api.Proc_wait) error {
	return p.Proc.Wait(ctx)
}

type processRegistry map[string]ProcessFactory

func (r processRegistry) Spawn(env *parens.Env, procType string, args []ww.Any) (Proc, error) {
	f, ok := r[procType]
	if !ok {
		return nil, parens.Error{
			Cause:   errors.New("no factory registered for process type"),
			Message: procType,
		}
	}

	return f(env, args)
}
