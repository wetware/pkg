package builtin

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
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
	case api.Value_Which_native:
		val = v
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
