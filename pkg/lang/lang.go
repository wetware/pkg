// Package lang contains the wetware language iplementation
package lang

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	"github.com/spy16/slurp"
	"github.com/spy16/slurp/core"
	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/proc"
	"github.com/wetware/ww/pkg/mem"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	globals = map[string]core.Any{
		"nil":   Nil{},
		"true":  True,
		"false": False,
	}
)

// ErrIncomparableTypes is returned if two types cannot be meaningfully
// compared to each other.
var ErrIncomparableTypes = errors.New("incomparable types")

// New returns a new root interpreter.
func New(a ww.Anchor) *slurp.Interpreter {
	if a == nil {
		panic("nil Anchor")
	}

	return slurp.New(
		slurp.WithAnalyzer(analyzer(a)),
		slurp.WithEnv(core.New(globals)))
}

// AsAny lifts a mem.Value to a ww.Any.
func AsAny(v mem.Value) (val ww.Any, err error) {
	switch v.Type() {
	case api.Value_Which_nil:
		val = Nil{}
	case api.Value_Which_bool:
		val = Bool{v}
	case api.Value_Which_i64:
		val = Int64{v}
	case api.Value_Which_f64:
		val = Float64{v}
	case api.Value_Which_bigInt:
		val, err = asBigInt(v)
	case api.Value_Which_bigFloat:
		val, err = asBigFloat(v)
	case api.Value_Which_frac:
		val, err = asFrac(v)
	case api.Value_Which_char:
		val = Char{v}
	case api.Value_Which_str:
		val = String{v}
	case api.Value_Which_keyword:
		val = Keyword{v}
	case api.Value_Which_symbol:
		val = Symbol{v}
	case api.Value_Which_path:
		val = Path{v}
	case api.Value_Which_list:
		val = list{v}
	case api.Value_Which_vector:
		val = vector{v}
	case api.Value_Which_proc:
		val = proc.FromValue(v)
	default:
		err = errors.Errorf("unknown value type '%s'", v.Type())
	}

	return
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

// Renderable values implement the Render method, which returns a human-readable
// representation of the data.
type Renderable interface {
	// Render returns a string representation of the value that is intended for human
	// consumption.  Strings returned by Render MAY be valid s-expressions, but this is
	// not guaranteed.
	Render() (string, error)
}

// SymbolProvider can provide a symbolic expression for a value.  Symbolic expressions
// are guaranteed to be parseable by a Reader.  Not all values can be represented as an
// s-expression.
type SymbolProvider interface {
	// SExpr returns the symbolic expression for the unerlying value.  Note that in the
	// case of composite values, the resulting s-expression can be very long.  For a
	// human-readable representation, see Renderable.
	SExpr() (string, error)
}

// Render a value into a human-readable representation.
// To serialize a value into a parseable s-expression, see SymbolProvider.
func Render(any ww.Any) (string, error) {
	switch v := any.(type) {
	case Renderable:
		return v.Render()
	case SymbolProvider:
		return v.SExpr()
	case fmt.Stringer:
		return v.String(), nil
	default:
		return fmt.Sprintf("%#v", v), nil
	}
}

// Hashable representation of an arbitrary value.
func Hashable(any ww.Any) ([]byte, error) {
	return capnp.Canonicalize(any.MemVal().Raw.Struct)
}

// Eq returns true is the two values are equal
func Eq(a, b ww.Any) (bool, error) {
	ba, err := Hashable(a)
	if err != nil {
		return false, err
	}

	bb, err := Hashable(b)
	if err != nil {
		return false, err
	}

	return bytes.Equal(ba, bb), nil
}

// Pop returns a collection without one item.  For a list, Pop
// returns a new list/queue without the first item, for a vector,
// returns a new vector without the last item. If the collection
// is empty, returns an error.
func Pop(col ww.Any) (ww.Any, error) {
	switch col.MemVal().Type() {
	case api.Value_Which_list:
		_, tail, err := listTail(col.MemVal())
		return tail, err

	case api.Value_Which_vector:
		_, vec, err := vectorPop(col.MemVal())
		return vec, err

	default:
		return nil, core.Error{
			Cause:   errors.New("unordered collection or atom"),
			Message: col.MemVal().Type().String(),
		}

	}
}

// Conj returns a new collection with the xs
// 'added'. (conj nil item) returns (item).  The 'addition' may
// happen at different 'places' depending on the concrete type.
func Conj(col ww.Any, vs core.Seq) (core.Any, error) {
	switch v := col.(type) {
	case List:
		err := core.ForEach(vs, func(item core.Any) (_ bool, err error) {
			v, err = v.Cons(item)
			return
		})
		return v, err

	case Vector:
		// TODO(performance): implement `v.Transient()`, returning *VectorBuilder.
		err := core.ForEach(vs, func(item core.Any) (_ bool, err error) {
			v, err = v.Conj(item.(ww.Any))
			return
		})
		return v, err

	default:
		return nil, core.Error{
			Cause:   errors.New("unordered collection or atom"),
			Message: col.MemVal().Type().String(),
		}

	}

}

// IsTruthy returns true if the value has a logical vale of `true`.
func IsTruthy(any ww.Any) (bool, error) {
	if any == nil {
		return false, nil
	}

	switch any.MemVal().Type() {
	case api.Value_Which_nil:
		return false, nil

	case api.Value_Which_bool:
		return any.MemVal().Raw.Bool(), nil

	case api.Value_Which_keyword, api.Value_Which_symbol, api.Value_Which_char, api.Value_Which_proc, api.Value_Which_path:
		return true, nil

	case api.Value_Which_str:
		s, err := any.MemVal().Raw.Str()
		if err != nil {
			return false, err
		}

		return len(s) > 0, nil

	case api.Value_Which_list:
		l, err := any.MemVal().Raw.List()
		if err != nil {
			return false, nil
		}

		return l.Count() > 0, nil

	case api.Value_Which_vector:
		vec, err := any.MemVal().Raw.Vector()
		if err != nil {
			return false, nil
		}

		return vec.Count() > 0, nil

	case api.Value_Which_i64:
		return any.MemVal().Raw.I64() != 0, nil

	case api.Value_Which_f64:
		return any.MemVal().Raw.F64() != 0, nil

	case api.Value_Which_bigInt:
		buf, err := any.MemVal().Raw.BigInt()
		if err != nil {
			return false, err
		}

		return len(buf) != 0, nil

	case api.Value_Which_bigFloat:
		return any.(BigFloat).f.Sign() == 0, nil

	case api.Value_Which_frac:
		return any.(Frac).r.Sign() == 0, nil

	default:
		if b, ok := any.(Boolable); ok {
			return b.Bool()
		}

		return false, core.Error{
			Cause: errors.Errorf("non-boolean type %s", reflect.TypeOf(any)),
		}
	}
}

// IsNil returns true if value is native go `nil` or `Nil{}`.
func IsNil(v core.Any) bool {
	if v == nil {
		return true
	}
	_, isNilType := v.(Nil)
	return isNilType
}
