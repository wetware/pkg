package builtin

import (
	"github.com/pkg/errors"
	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/proc"
	"github.com/wetware/ww/pkg/mem"
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
		val = proc.FromValue(v)
	default:
		err = errors.Errorf("unknown value type '%s'", v.Type())
	}

	return
}
