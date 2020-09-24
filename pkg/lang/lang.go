// Package lang contains the wetware language iplementation
package lang

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/spy16/parens"
	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/proc"
	"github.com/wetware/ww/pkg/mem"
)

var (
	globals = map[string]parens.Any{
		"nil":   Nil{},
		"true":  True,
		"false": False,
	}
)

// ErrIncomparableTypes is returned if two types cannot be meaningfully
// compared to each other.
var ErrIncomparableTypes = errors.New("incomparable types")

// New returns a new root environment.
func New(a ww.Anchor) *parens.Env {
	if a == nil {
		panic("nil Anchor")
	}

	return parens.New(
		parens.WithAnalyzer(analyzer(a)),
		parens.WithGlobals(globals, nil))
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
		val = List{v}
	case api.Value_Which_vector:
		val = Vector{v}
	case api.Value_Which_proc:
		val = proc.Proc{Value: v}
	default:
		err = errors.Errorf("unknown value type '%s'", v.Type())
	}

	return
}

// Comparable type.
type Comparable interface {
	// Comp compares the magnitude of the comparable c with that of other.
	// It returns 0 if the magnitudes are equal, -1 if c < other, and 1 if c > other.
	Comp(other ww.Any) (int, error)
}

// Eq returns true is the two values are equal
func Eq(a, b ww.Any) bool {
	return bytes.Equal(a.MemVal().Bytes(), b.MemVal().Bytes())
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
		return nil, parens.Error{
			Cause:   errors.New("unordered collection or atom"),
			Message: col.MemVal().Type().String(),
		}

	}
}

// Conj returns a new collection with the xs
// 'added'. (conj nil item) returns (item).  The 'addition' may
// happen at different 'places' depending on the concrete type.
func Conj(col ww.Any, vs parens.Seq) (parens.Any, error) {
	switch col.MemVal().Type() {
	case api.Value_Which_list:
		l := List{col.MemVal()}
		err := parens.ForEach(vs, func(v parens.Any) (_ bool, err error) {
			l, err = l.Cons(v)
			return
		})
		return l, err

	case api.Value_Which_vector:
		vec := Vector{col.MemVal()}
		err := parens.ForEach(vs, func(v parens.Any) (_ bool, err error) {
			vec, err = vec.Conj(v)
			return
		})
		return vec, err

	default:
		return nil, parens.Error{
			Cause:   errors.New("unordered collection or atom"),
			Message: col.MemVal().Type().String(),
		}

	}
}
