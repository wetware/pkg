package builtin

import (
	"fmt"

	"github.com/wetware/ww/pkg/lang/core"
	"github.com/wetware/ww/pkg/mem"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	_ core.Nil     = (*Nil)(nil)
	_ core.Bool    = (*Bool)(nil)
	_ core.String  = (*String)(nil)
	_ core.Keyword = (*Keyword)(nil)
	_ core.Symbol  = (*Symbol)(nil)
	_ core.Char    = (*Char)(nil)

	// True value of Bool
	True Bool

	//False value of Bool
	False Bool
)

func init() {
	var err error
	if True, err = NewBool(capnp.SingleSegment(nil), true); err != nil {
		panic(err)
	}

	if False, err = NewBool(capnp.SingleSegment(nil), false); err != nil {
		panic(err)
	}
}

// Nil represents a null value.
type Nil struct{}

// Nil satisfies core.Nil.
func (Nil) Nil() {}

func (Nil) String() string { return "nil" }

// MemVal returns the memory value.
func (Nil) MemVal() mem.Value { return mem.NilValue }

// Bool represents a boolean value.
type Bool struct{ mem.Value }

// NewBool .
func NewBool(a capnp.Arena, b bool) (Bool, error) {
	val, err := mem.NewValue(a)
	if err == nil {
		val.Raw.SetBool(b)
	}

	return Bool{val}, err
}

// Bool satisfies core.Bool.
func (b Bool) Bool() bool { return b.Raw.Bool() }

func (b Bool) String() string {
	if b.Bool() {
		return "true"
	}
	return "false"
}

// Char represents a character literal.  For example, \a, \b, \1, \∂ etc are
// valid character literals. In addition, special literals like \newline, \space
// etc are supported by the reader.
type Char struct{ mem.Value }

// NewChar .
func NewChar(a capnp.Arena, r rune) (Char, error) {
	val, err := mem.NewValue(a)
	if err == nil {
		val.Raw.SetChar(r)
	}

	return Char{val}, err
}

// Char satisfies core.Char.
func (c Char) Char() rune { return c.Raw.Char() }

func (c Char) String() string { return fmt.Sprintf("\\%c", c.Char()) }

// String represents text. Escape sequences are not applicable at this level.
type String struct{ mem.Value }

// NewString .
func NewString(a capnp.Arena, s string) (String, error) {
	val, err := mem.NewValue(a)
	if err == nil {
		err = val.Raw.SetStr(s)
	}

	return String{val}, err
}

func (str String) String() (s string, err error) {
	if s, err = str.Raw.Str(); err == nil {
		s = "\"" + s + "\""
	}

	return
}

// Render the string into a parseable s-expression.
func (str String) Render() (string, error) { return str.String() }

// Keyword represents a keyword literal.
type Keyword struct{ mem.Value }

// NewKeyword .
func NewKeyword(a capnp.Arena, s string) (Keyword, error) {
	val, err := mem.NewValue(a)
	if err == nil {
		err = val.Raw.SetKeyword(s)
	}

	return Keyword{val}, err
}

// Keyword satisfies core.Keyword.
func (kw Keyword) Keyword() (string, error) { return kw.Raw.Keyword() }

// Render the keyword into a parseable s-expression.
func (kw Keyword) Render() (string, error) {
	s, err := kw.Keyword()
	if err != nil {
		return "", err
	}

	return ":" + s, nil
}

// Symbol represents a name given to a value in memory.
type Symbol struct{ mem.Value }

// NewSymbol .
func NewSymbol(a capnp.Arena, s string) (Symbol, error) {
	val, err := mem.NewValue(a)
	if err == nil {
		err = val.Raw.SetSymbol(s)
	}

	return Symbol{val}, err
}

// Symbol satisfies core.Symbol.
func (sym Symbol) Symbol() (string, error) { return sym.Raw.Symbol() }

// Render the symbol into a parseable s-expression.
func (sym Symbol) Render() (string, error) { return sym.Symbol() }
