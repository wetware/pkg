package core

import (
	"context"
	"errors"

	"github.com/spy16/slurp/core"
	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
)

var (
	registry = processRegistry{}

	_ ww.Any = (Process)(nil)
)

// ProcessFactory constructs a process from arguments, and starts it in the
// background.
type ProcessFactory func(env core.Env, args []ww.Any) (Process, error)

// RegisterProcessFactory a process implementation by mapping
// a process type to a ProcessFactory.".
//
// If f is nil, the corresponding tag is deleted from the registry.
// Duplicate factories for a given process type will be overwritten.
func RegisterProcessFactory(procType string, f ProcessFactory) {
	if f == nil {
		delete(registry, procType)
	} else {
		registry[procType] = f
	}
}

// Process is a generic asynchronous process
type Process interface {
	ww.Any
	Wait(context.Context) error
}

// Spawn configures a process based on the supplied arguments and then starts it.
func Spawn(env core.Env, args ...ww.Any) (Process, error) {
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

type processRegistry map[string]ProcessFactory

func (r processRegistry) Spawn(env core.Env, procType string, args []ww.Any) (Process, error) {
	f, ok := r[procType]
	if !ok {
		return nil, core.Error{
			Cause:   errors.New("no factory registered for process type"),
			Message: procType,
		}
	}

	return f(env, args)
}
