package core

import (
	"errors"
	"fmt"

	"github.com/spy16/slurp/core"
	ww "github.com/wetware/ww/pkg"
)

var (
	// ErrIncomparableTypes is returned if two types cannot be meaningfully
	// compared to each other.
	ErrIncomparableTypes = errors.New("incomparable types")

	// ErrIndexOutOfBounds is returned when a sequence's index is out of range.
	ErrIndexOutOfBounds = errors.New("index out of bounds")

	// ErrNotFound is returned by Env when a binding is not found for a given symbol/name.
	ErrNotFound = core.ErrNotFound

	// ErrArity is returned when an Invokable is invoked with wrong number
	// of arguments.
	ErrArity = core.ErrArity

	// ErrNotInvokable is returned by InvokeExpr when the target is not invokable.
	ErrNotInvokable = core.ErrNotInvokable
)

type (
	// Env represents the environment in which forms are evaluated.
	Env = core.Env

	// Analyzer implementation is responsible for performing syntax analysis
	// on given form.
	Analyzer = core.Analyzer

	// Expr represents an expression that can be evaluated against an env.
	Expr = core.Expr

	// Error is returned by all slurp operations. Cause indicates the underlying
	// error type. Use errors.Is() with Cause to check for specific errors.
	Error = core.Error
)

// New returns a root Env that can be used to execute forms.
func New(globals map[string]ww.Any) Env {
	gs := make(map[string]core.Any, len(globals))
	for symbol, value := range globals {
		gs[symbol] = value
	}

	return core.New(gs)
}

// Boolable values can be evaluated as true or false
type Boolable interface {
	Bool() (bool, error)
}

// Comparable type.
type Comparable interface {
	// Comp compares the magnitude of the comparable c with that of other.
	// It returns 0 if the magnitudes are equal, -1 if c < other, and 1 if c > other.
	Comp(other ww.Any) (int, error)
}

// EqualityProvider can test for equality.
type EqualityProvider interface {
	Eq(ww.Any) (bool, error)
}

// Renderable types provide a human-readable representation.
type Renderable interface {
	Render() (string, error)
}

// Render a value into a human-readable representation.
// To serialize a value into a parseable s-expression, see core.SExpressable.
func Render(any ww.Any) (string, error) {
	switch v := any.(type) {
	case Renderable:
		return v.Render()
	case fmt.Stringer:
		return v.String(), nil
	default:
		return fmt.Sprintf("%#v", v), nil
	}
}
