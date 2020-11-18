package builtin

import (
	"bytes"
	"reflect"

	"github.com/pkg/errors"
	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/core"
	"github.com/wetware/ww/pkg/mem"
	capnp "zombiezen.com/go/capnproto2"
)

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
		val = RemoteProcess{v}
	default:
		err = errors.Errorf("unknown value type '%s'", v.Type())
	}

	return
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
func Conj(col ww.Any, vs core.Seq) (ww.Any, error) {
	switch v := col.(type) {
	case core.List:
		err := core.ForEach(vs, func(item ww.Any) (_ bool, err error) {
			v, err = v.Cons(item)
			return
		})
		return v, err

	case core.Vector:
		// TODO(performance): implement `v.Transient()`, returning *VectorBuilder.
		err := core.ForEach(vs, func(item ww.Any) (_ bool, err error) {
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
		if b, ok := any.(core.Bool); ok {
			return b.Bool(), nil
		}

		return false, core.Error{
			Cause: errors.Errorf("non-boolean type %s", reflect.TypeOf(any)),
		}
	}
}

// IsNil returns true if value is native go `nil` or `Nil{}`.
func IsNil(v ww.Any) bool {
	if v == nil {
		return true
	}

	return v.MemVal().Nil()
}
