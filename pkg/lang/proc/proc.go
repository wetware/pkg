// Package proc provides a plugin architecture for Wetware proceses.
package proc

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spy16/parens"
	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/mem"
)

var (
	registry = processRegistry{}

	_ ww.Any = (Proc)(nil)
)

// ProcessFactory constructs a process from arguments, and starts it in the
// background.
type ProcessFactory func(env *parens.Env, args []ww.Any) (Proc, error)

// Register a process implementation by mapping a process type to a ProcessFactory.
// Types MUST follow keyword syntax, e.g. ":go" or ":unix".
// If f is nil, the corresponding tag is deleted from the registry.
// Duplicate factories for a given process type will be overwritten.
func Register(procType string, f ProcessFactory) {
	if err := validateProcType(procType); err != nil {
		panic(err)
	}

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
	if len(args) == 0 {
		return nil, errors.New("expected at least 1 argument, got 0")
	}

	tag, err := args[0].SExpr()
	if err != nil {
		return nil, err
	}

	if err := validateProcType(tag); err != nil {
		return nil, err
	}

	return registry.Spawn(env, tag, args[1:])
}

// FromValue lifts a mem.Value into a first-class Proc type.
// It performs no type-checking, so callers MUST ensure v is a valid Proc.
func FromValue(v mem.Value) Proc { return remoteProc{v} }

type remoteProc struct{ mem.Value }

func (p remoteProc) SExpr() (string, error) {
	// TODO(performance):  we should probably fetch the SExpr from the remote cap _once_
	// 					   and then cache it locally.

	return "remoteProc.SExpr() NOT IMPLEMENTED", nil
}

func (p remoteProc) Wait(ctx context.Context) error {
	_, err := p.Raw.Proc().Wait(ctx, func(api.Proc_wait_Params) error { return nil }).Struct()
	return err
}

type procCap struct{ Proc }

func (p procCap) Wait(call api.Proc_wait) error { return p.Proc.Wait(call.Ctx) }

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

func validateProcType(procType string) error {
	if procType[0] != ':' {
		return parens.Error{
			Cause:   errors.New("invalid process type"),
			Message: procType,
		}
	}

	// procType is ":" ?
	if len(procType) == 1 {
		return parens.Error{Cause: errors.New("empty process type")}
	}

	return nil
}
