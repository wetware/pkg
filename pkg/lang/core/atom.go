package core

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spy16/sabre/runtime"
	"github.com/wetware/ww/internal/api"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	_ runtime.Value = (*Bool)(nil)
	_ runtime.Value = (*String)(nil)
	_ runtime.Value = (*Keyword)(nil)
	_ runtime.Value = (*Symbol)(nil)
	_ runtime.Value = (*Char)(nil)
	// _ runtime.Value = Int64(0)
	// _ runtime.Value = Float64(0)

	_ runtime.Invokable = (*Keyword)(nil)

	_ apiValueProvider = (*Bool)(nil)
	_ apiValueProvider = (*String)(nil)
	_ apiValueProvider = (*Keyword)(nil)
	_ apiValueProvider = (*Symbol)(nil)
	_ apiValueProvider = (*Char)(nil)
)

func valueOf(v api.Value) (val runtime.Value, err error) {
	switch w := v.Which(); w {
	case api.Value_Which_nil:
		val = Nil{}
	case api.Value_Which_bool:
		val = Bool{v: v}
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
type Nil = runtime.Nil

// Bool represents a boolean value.
type Bool struct {
	v api.Value
}

// NewBool .
func NewBool(b bool) (bl Bool, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(capnp.SingleSegment(nil)); err != nil {
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

// Eval returns the underlying value.
func (b Bool) Eval(runtime.Runtime) (runtime.Value, error) {
	return b, nil
}

// Equals returns true if 'other' is a boolean and has same logical value.
func (b Bool) Equals(other runtime.Value) bool {
	o, ok := other.(Bool)
	return ok && (o.v.Bool() == b.v.Bool())
}

func (b Bool) String() string {
	return fmt.Sprintf("%t", b.v.Bool())
}

// Char represents a character literal.  For example, \a, \b, \1, \∂ etc are
// valid character literals. In addition, special literals like \newline, \space
// etc are supported by the reader.
type Char struct {
	v api.Value
}

// NewChar .
func NewChar(r rune) (c Char, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(capnp.SingleSegment(nil)); err != nil {
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

// Eval simply returns itself since Chracters evaluate to themselves.
func (c Char) Eval(runtime.Runtime) (runtime.Value, error) {
	return c, nil
}

// Equals returns true if the other value is also a character and has same value.
func (c Char) Equals(other runtime.Value) bool {
	o, isChar := other.(Char)
	return isChar && (o.v.Char() == c.v.Char())
}

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
func NewString(s string) (str String, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(capnp.SingleSegment(nil)); err != nil {
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

// Eval simply returns itself since Strings evaluate to themselves.
func (s String) Eval(runtime.Runtime) (runtime.Value, error) {
	return s, nil
}

// Equals returns true if 'other' is string and has same value.
func (s String) Equals(other runtime.Value) bool {
	val, ok := other.(String)
	return ok && (val.string() == s.string())
}

func (s String) String() string {
	return fmt.Sprintf("\"%s\"", s.string())
}

func (s String) string() string {
	str, err := s.v.Str()
	if err != nil {
		panic(err)
	}

	return str
}

// Keyword represents a keyword literal.
type Keyword struct {
	v api.Value
}

// NewKeyword .
func NewKeyword(s string) (kw Keyword, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(capnp.SingleSegment(nil)); err != nil {
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

// Eval simply returns itself since Keywords evaluate to themselves.
func (kw Keyword) Eval(runtime.Runtime) (runtime.Value, error) {
	return kw, nil
}

// Equals returns true if the other value is keyword and has same value.
func (kw Keyword) Equals(other runtime.Value) bool {
	val, isKeyword := other.(Keyword)
	return isKeyword && (val.string() == kw.string())
}

func (kw Keyword) String() string {
	return fmt.Sprintf(":%s", kw.string())
}

// Invoke map lookups.
func (kw Keyword) Invoke(scope runtime.Runtime, args ...runtime.Value) (runtime.Value, error) {
	if len(args) != 1 && len(args) != 2 {
		return nil, fmt.Errorf("keyword invokation expects 1 or 2 arguments, got %d", len(args))
	}

	argVals, err := runtime.EvalAll(scope, args)
	if err != nil {
		return nil, err
	}

	assocVal, ok := argVals[0].(runtime.Map)
	if !ok {
		return Nil{}, nil
	}

	var def runtime.Value = Nil{}
	if len(argVals) == 2 {
		def = argVals[1]
	}

	val := assocVal.EntryAt(kw)
	if val == nil {
		val = def
	}

	return val, nil
}

func (kw Keyword) string() string {
	str, err := kw.v.Keyword()
	if err != nil {
		panic(err)
	}

	return str
}

// Symbol represents a name given to a value in memory.
type Symbol struct {
	runtime.Position
	v api.Value
}

// NewSymbol .
func NewSymbol(s string) (sym Symbol, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(capnp.SingleSegment(nil)); err != nil {
		return
	}

	if sym.v, err = api.NewRootValue(seg); err == nil {
		err = sym.v.SetSymbol(s)
	}

	return
}

// Value for Symbol type
func (s Symbol) Value() api.Value {
	return s.v
}

// Eval returns the value bound to this symbol in current context.
func (s Symbol) Eval(scope runtime.Runtime) (runtime.Value, error) {
	return scope.Resolve(s.string())
}

// Equals returns true if the other value is also a symbol and has same value.
func (s Symbol) Equals(other runtime.Value) bool {
	val, isSym := other.(Symbol)
	return isSym && (s.string() == val.string())
}

func (s Symbol) String() string {
	return s.string()
}

func (s Symbol) string() string {
	str, err := s.v.Symbol()
	if err != nil {
		panic(err)
	}

	return str
}
