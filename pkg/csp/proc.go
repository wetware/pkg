package csp

import (
	"fmt"

	"capnproto.org/go/capnp/v3"
	"github.com/wetware/ww/internal/api/proc"
)

// Param represents a single parameter in a process' execution context.
type Param[T ~capnp.StructKind] func(T) error

// Type is a type parameter that is passed to NewConfig to create
// a Context for a specific Executor implementation.
type Type[T ~capnp.StructKind] func(*capnp.Segment) (T, error)

// Context is a generic 'SetParams' callback that can be passed to a generic
// exec call.
type Context[T ~capnp.StructKind] func(proc.Executor_exec_Params) error

func NewContext[T ~capnp.StructKind](alloc Type[T]) Context[T] {
	return func(ps proc.Executor_exec_Params) error {
		t, err := alloc(ps.Segment())
		if err != nil {
			return fmt.Errorf("alloc config: %w", err)
		}

		return ps.SetConfig(config(t))
	}
}

// Bind a parameter to the configuration type.
func (c Context[T]) Bind(param Param[T]) Context[T] {
	return func(ps proc.Executor_exec_Params) error {
		if c != nil {
			if err := c(ps); err != nil {
				return err
			}
		}

		return bind(ps, param)
	}
}

func bind[T ~capnp.StructKind](ps proc.Executor_exec_Params, param Param[T]) error {
	ptr, err := ps.Config()
	if err != nil {
		return err
	}

	return param(T(ptr.Struct()))
}

func config[T ~capnp.StructKind](t T) capnp.Ptr {
	return capnp.Struct(t).ToPtr()
}
