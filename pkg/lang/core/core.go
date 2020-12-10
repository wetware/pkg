package core

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"

	"github.com/spy16/slurp/core"
	"github.com/wetware/ww/internal/mem"
	ww "github.com/wetware/ww/pkg"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	// ErrIncomparableTypes is returned if two types cannot be meaningfully
	// compared to each other.
	ErrIncomparableTypes = errors.New("incomparable types")

	// ErrIndexOutOfBounds is returned when a sequence's index is out of range.
	ErrIndexOutOfBounds = errors.New("index out of bounds")

	// ErrNotFound is returned by Env when a the corresponding entity for a name,
	// binding or module path is not found.
	ErrNotFound = core.ErrNotFound

	// ErrArity is returned when an Invokable is invoked with wrong number
	// of arguments.
	ErrArity = core.ErrArity

	// ErrNotInvokable is returned by InvokeExpr when the target is not invokable.
	ErrNotInvokable = core.ErrNotInvokable

	// ErrIllegalState is returned when an operation is performed against a correct
	// type with an invalid value.
	ErrIllegalState = errors.New("illegal state")

	errType = reflect.TypeOf((*error)(nil)).Elem()
	anyType = reflect.TypeOf((*ww.Any)(nil)).Elem()
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

// Eval a form.
func Eval(env Env, a Analyzer, form core.Any) (core.Any, error) {
	return core.Eval(env, a, form)
}

// Invokable represents a value that can be invoked as a function.
type Invokable interface {
	// Invoke is called if this value appears as the first argument of
	// invocation form (i.e., list).
	Invoke(args ...ww.Any) (ww.Any, error)
}

// Countable types can report the number of elements they contain.
type Countable interface {
	// Count provides the number of elements contained.
	Count() (int, error)
}

// Container is an aggregate of values.
type Container interface {
	ww.Any
	Countable
	Conj(...ww.Any) (Container, error)
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

	return v.Value().Which() == mem.Any_Which_nil
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

	case Countable:
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
	if a.Value().Which() == b.Value().Which() {
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

// Pop an item from an ordered collection.
// For a list, returns a new list without the first item.
// For a vector, returns a new vector without the last item.
// If the collection is empty, returns a wrapped ErrIllegalState.
func Pop(cont Container) (ww.Any, error) {
	switch v := cont.(type) {
	case Vector:
		return v.Pop()

	case Seq:
		cnt, err := v.Count()
		if err != nil {
			return nil, err
		}

		if cnt == 0 {
			return nil, fmt.Errorf("%w: cannot pop from empty seq", ErrIllegalState)
		}

		return v.Next()

	}

	return nil, fmt.Errorf("cannot pop from %s", cont.Value().Which())
}

// Conj returns a new collection with the supplied
// values "conjoined".
//
// For lists, the value is added at the head.
// For vectors, the value is added at the tail.
// `(conj nil item)` returns `(item)``.
func Conj(any ww.Any, xs ...ww.Any) (Container, error) {
	if IsNil(any) {
		return NewList(capnp.SingleSegment(nil), xs...)
	}

	if c, ok := any.(Container); ok {
		return c.Conj(xs...)
	}

	return nil, fmt.Errorf("cannot conj with %T", any)
}

// Canonical representation of an arbitrary value.
func Canonical(any ww.Any) ([]byte, error) {
	return capnp.Canonicalize(any.Value().Struct)
}

// AsAny lifts a mem.Any to a ww.Any.
func AsAny(any mem.Any) (val ww.Any, err error) {
	switch any.Which() {
	case mem.Any_Which_nil:
		val = Nil{}
	case mem.Any_Which_bool:
		val = Bool{any}
	case mem.Any_Which_i64:
		val = i64{any}
	case mem.Any_Which_f64:
		val = f64{any}
	case mem.Any_Which_bigInt:
		val, err = asBigInt(any)
	case mem.Any_Which_bigFloat:
		val, err = asBigFloat(any)
	case mem.Any_Which_frac:
		val, err = asFrac(any)
	case mem.Any_Which_char:
		val = Char{any}
	case mem.Any_Which_str:
		val = String{any}
	case mem.Any_Which_keyword:
		val = Keyword{any}
	case mem.Any_Which_symbol:
		val = Symbol{any}
	case mem.Any_Which_path:
		val = Path{any}
	case mem.Any_Which_list:
		val = list{any}
	case mem.Any_Which_vector:
		val = DeepPersistentVector{any}
	// case mem.Any_Which_proc:
	// 	val = RemoteProcess{v}
	default:
		err = fmt.Errorf("unknown value type '%s'", any.Which())
	}

	return
}
