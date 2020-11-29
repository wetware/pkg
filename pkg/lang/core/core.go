package core

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/spy16/slurp/core"
	ww "github.com/wetware/ww/pkg"
	capnp "zombiezen.com/go/capnproto2"
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
// It binds the prelude to the environment before returning.
func New() Env { return core.New(nil) }

// Invokable represents a value that can be invoked as a function.
type Invokable interface {
	// Invoke is called if this value appears as the first argument of
	// invocation form (i.e., list).
	Invoke(args ...ww.Any) (ww.Any, error)
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
func Render(v ww.Any) (string, error) {
	switch val := v.(type) {
	case Renderable:
		return val.Render()
	case fmt.Stringer:
		return val.String(), nil
	default:
		return fmt.Sprintf("%#v", val), nil
	}
}

// IsNil returns true if value is native go `nil` or `Nil{}`.
func IsNil(v ww.Any) bool {
	if v == nil {
		return true
	}

	return v.MemVal().Nil()
}

// IsTruthy returns true if the value has a logical vale of `true`.
func IsTruthy(v ww.Any) (bool, error) {
	if IsNil(v) {
		return false, nil
	}

	switch val := v.(type) {
	case Bool:
		return val.Bool(), nil

	case Numerical:
		return !val.Zero(), nil

	case interface{ Count() (int, error) }: // container types
		i, err := val.Count()
		return i == 0, err

	default:
		return true, nil

	}
}

// Eq returns true is the two values are equal
func Eq(a, b ww.Any) (bool, error) {
	// Nil is only equal to itself
	if IsNil(a) && IsNil(b) {
		return true, nil
	}

	// Check for usable interfaces on object A
	switch val := a.(type) {
	case Comparable:
		i, err := val.Comp(b)
		return i == 0, err

	case EqualityProvider:
		return val.Eq(b)

	}

	// Check for usable interfaces on object B
	switch val := b.(type) {
	case Comparable:
		i, err := val.Comp(b)
		return i == 0, err

	case EqualityProvider:
		return val.Eq(b)

	}

	// Identical types with the same canonical representation are equal.
	if a.MemVal().Type() == b.MemVal().Type() {
		ca, err := Canonical(a)
		if err != nil {
			return false, err
		}

		cb, err := Canonical(b)
		if err != nil {
			return false, err
		}

		return bytes.Equal(ca, cb), nil
	}

	// Disparate types are unequal by default.
	return false, nil
}

// // Sum of numerical values.
// func Sum(ns ...Numerical) (n Numerical, err error) {
// 	if len(ns) == 0 {

// 	}

// 	n, ns := ns[0], ns[1:]
// }

// Canonical representation of an arbitrary value.
func Canonical(any ww.Any) ([]byte, error) {
	return capnp.Canonicalize(any.MemVal().Raw.Struct)
}
