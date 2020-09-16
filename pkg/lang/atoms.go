package lang

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spy16/parens"
	"github.com/wetware/ww/internal/api"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	_ parens.Any = (*Bool)(nil)
	_ parens.Any = (*String)(nil)
	_ parens.Any = (*Keyword)(nil)
	_ parens.Any = (*Symbol)(nil)
	_ parens.Any = (*Char)(nil)

	_ apiValueProvider = (*Bool)(nil)
	_ apiValueProvider = (*String)(nil)
	_ apiValueProvider = (*Keyword)(nil)
	_ apiValueProvider = (*Symbol)(nil)
	_ apiValueProvider = (*Char)(nil)
)

func valueOf(v api.Value) (val parens.Any, err error) {
	switch w := v.Which(); w {
	case api.Value_Which_nil:
		val = Nil{}
	case api.Value_Which_bool:
		val = Bool{v: v}
	case api.Value_Which_i64:
		val = Int64{v: v}
	case api.Value_Which_f64:
		val = Float64{v: v}
	case api.Value_Which_bigInt:
		val, err = bigIntFromValue(v)
	case api.Value_Which_bigFloat:
		val, err = bigFloatFromValue(v)
	case api.Value_Which_frac:
		val, err = fracFromValue(v)
	case api.Value_Which_char:
		val = Char{v: v}
	case api.Value_Which_str:
		if v.HasStr() {
			val = String{v: v}
		} else {
			err = enoval(w)
		}
	case api.Value_Which_keyword:
		if v.HasKeyword() {
			val = Keyword{v: v}
		} else {
			err = enoval(w)
		}
	case api.Value_Which_symbol:
		if v.HasSymbol() {
			val = Symbol{v: v}
		} else {
			err = enoval(w)
		}
	case api.Value_Which_path:
		if v.HasPath() {
			val = Path{v: v}
		} else {
			err = enoval(w)
		}
	case api.Value_Which_list:
		if v.HasList() {
			val = List{v: v}
		} else {
			err = enoval(w)
		}
	case api.Value_Which_vector:
		if v.HasVector() {
			val = Vector{v: v}
		} else {
			err = enoval(w)
		}
	default:
		panic(errors.Errorf("unknown value type '%s'", w))
	}

	return
}

func enoval(w api.Value_Which) error {
	return errors.Errorf("ValueError: missing %s", w)
}

// Nil value
type Nil = parens.Nil

// Bool represents a boolean value.
type Bool struct {
	v api.Value
}

// NewBool .
func NewBool(a capnp.Arena, b bool) (bl Bool, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if bl.v, err = api.NewRootValue(seg); err == nil {
		bl.v.SetBool(b)
	}

	return
}

// Value for Bool type
func (b Bool) Value() api.Value {
	return b.v
}

// SExpr returns a valid s-expression representing Bool.
func (b Bool) SExpr() (string, error) { return b.String(), nil }

func (b Bool) String() string {
	if b.v.Bool() {
		return "true"
	}
	return "false"
}

// // Eq returns true if 'other' is a boolean and has same logical value.
// func (b Bool) Eq(other parens.Any) bool {
// 	o, ok := other.(Bool)
// 	return ok && (o.v.Bool() == b.v.Bool())
// }

// Char represents a character literal.  For example, \a, \b, \1, \∂ etc are
// valid character literals. In addition, special literals like \newline, \space
// etc are supported by the reader.
type Char struct {
	v api.Value
}

// NewChar .
func NewChar(a capnp.Arena, r rune) (c Char, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if c.v, err = api.NewRootValue(seg); err == nil {
		c.v.SetChar(r)
	}

	return
}

// Value for the Char type
func (c Char) Value() api.Value {
	return c.v
}

// SExpr returns a valid s-expression representing Char.
func (c Char) SExpr() (string, error) {
	return fmt.Sprintf("\\%c", c.v.Char()), nil
}

// // Eq returns true if the other value is also a character and has same value.
// func (c Char) Eq(other parens.Any) bool {
// 	o, isChar := other.(Char)
// 	return isChar && (o.v.Char() == c.v.Char())
// }

func (c Char) String() string {
	return fmt.Sprintf("\\%c", rune(c.v.Char()))
}

// String represents double-quoted string literals. String Form represents
// the true string value obtained from the reader. Escape sequences are not
// applicable at this level.
type String struct {
	v api.Value
}

// NewString .
func NewString(a capnp.Arena, s string) (str String, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if str.v, err = api.NewRootValue(seg); err == nil {
		err = str.v.SetStr(s)
	}

	return
}

// Value for String type
func (s String) Value() api.Value {
	return s.v
}

// SExpr returns a valid s-expression representing String.
func (s String) SExpr() (str string, err error) {
	if str, err = s.v.Str(); err == nil {
		str = "\"" + str + "\""
	}

	return
}

// // Eq returns true if 'other' is string and has same value.
// func (s String) Eq(other parens.Any) bool {
// 	str, err := s.v.Str()
// 	if err != nil {
// 		panic(err)
// 	}

// 	o, ok := other.(String)
// 	if !ok {
// 		return false
// 	}

// 	otherStr, err := o.v.Str()
// 	if err != nil {
// 		panic(err) // TODO(upstream):  return error from Eq()
// 	}

// 	return str == otherStr
// }

// Keyword represents a keyword literal.
type Keyword struct {
	v api.Value
}

// NewKeyword .
func NewKeyword(a capnp.Arena, s string) (kw Keyword, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if kw.v, err = api.NewRootValue(seg); err == nil {
		err = kw.v.SetKeyword(s)
	}

	return
}

// Value for Keyword type
func (kw Keyword) Value() api.Value {
	return kw.v
}

// SExpr returns a valid s-expression for the keyword
func (kw Keyword) SExpr() (string, error) {
	s, err := kw.v.Keyword()
	if err != nil {
		return "", err
	}

	return ":" + s, nil
}

// // Eq returns true if the other value is keyword and has same value.
// func (kw Keyword) Eq(other parens.Any) bool {
// 	keyword, err := kw.v.Keyword()
// 	if err != nil {
// 		panic(err)
// 	}

// 	o, ok := other.(Keyword)
// 	if !ok {
// 		return false
// 	}

// 	otherKW, err := o.v.Keyword()
// 	if err != nil {
// 		panic(err) // TODO(upstream):  return error from Eq()
// 	}

// 	return keyword == otherKW
// }

// Symbol represents a name given to a value in memory.
type Symbol struct {
	v api.Value
}

// NewSymbol .
func NewSymbol(a capnp.Arena, s string) (sym Symbol, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if sym.v, err = api.NewRootValue(seg); err == nil {
		err = sym.v.SetSymbol(s)
	}

	return
}

// SExpr returns a valid s-expression for the symbol
func (s Symbol) SExpr() (string, error) {
	sym, err := s.v.Symbol()
	if err != nil {
		return "", err
	}

	return sym, nil
}

// Value for Symbol type
func (s Symbol) Value() api.Value {
	return s.v
}

// // Eq returns true if the other value is also a symbol and has same value.
// func (s Symbol) Eq(other parens.Any) bool {
// 	sym, err := s.v.Symbol()
// 	if err != nil {
// 		panic(err)
// 	}

// 	o, ok := other.(Symbol)
// 	if !ok {
// 		return false
// 	}

// 	otherSym, err := o.v.Symbol()
// 	if err != nil {
// 		panic(err) // TODO(upstream):  return error from Eq()
// 	}

// 	return sym == otherSym
// }
