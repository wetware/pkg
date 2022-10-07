package process

import (
	"fmt"

	"capnproto.org/go/capnp/v3"
	"github.com/wetware/ww/internal/api/proc"
)

// Param represents a single parameter in a configuration type.
// TODO:  create capnp.PtrKind and update type constraint.
type Param[T ~capnp.StructKind] func(T) error

// ConfigType can allocate a configuration struct for the specific Executor
// implementation.
type ConfigType[T ~capnp.StructKind] func(capnp.Arena) (T, error)

// Config is a generic 'SetParams' callback that can be passed to a generic
// exec call.
type Config[T ~capnp.StructKind] func(proc.Executor_exec_Params) error

func NewConfig[T ~capnp.StructKind](alloc ConfigType[T]) Config[T] {
	return func(ps proc.Executor_exec_Params) error {
		t, err := alloc(ps.Message().Arena)
		if err != nil {
			return fmt.Errorf("alloc config: %w", err)
		}

		return ps.SetConfig(config(t))
	}
}

// Bind a parameter to the configuration type.
func (c Config[T]) Bind(param Param[T]) Config[T] {
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
