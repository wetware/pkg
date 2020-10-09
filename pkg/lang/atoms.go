package lang

import (
	"fmt"

	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/mem"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	_ ww.Any = (*Bool)(nil)
	_ ww.Any = (*String)(nil)
	_ ww.Any = (*Keyword)(nil)
	_ ww.Any = (*Symbol)(nil)
	_ ww.Any = (*Char)(nil)

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

func (Nil) String() string { return "nil" }

// SExpr returns a valid s-expression for Nil
func (Nil) SExpr() (string, error) { return "nil", nil }

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

func (b Bool) String() string {
	if b.Raw.Bool() {
		return "true"
	}
	return "false"
}

// SExpr returns a valid s-expression for Bool
func (b Bool) SExpr() (string, error) { return b.String(), nil }

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

func (c Char) String() string { return fmt.Sprintf("\\%c", c.Raw.Char()) }

// SExpr returns a valid s-expression for Char
func (c Char) SExpr() (string, error) { return c.String(), nil }

// String represents double-quoted string literals. String Form represents
// the true string value obtained from the reader. Escape sequences are not
// applicable at this level.
type String struct{ mem.Value }

// NewString .
func NewString(a capnp.Arena, s string) (String, error) {
	val, err := mem.NewValue(a)
	if err == nil {
		err = val.Raw.SetStr(s)
	}

	return String{val}, err
}

// SExpr returns a valid s-expression representing String.
func (str String) SExpr() (s string, err error) {
	if s, err = str.Raw.Str(); err == nil {
		s = "\"" + s + "\""
	}

	return
}

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

// SExpr returns a valid s-expression for the keyword
func (kw Keyword) SExpr() (string, error) {
	s, err := kw.Raw.Keyword()
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

// SExpr returns a valid s-expression for the symbol
func (sym Symbol) SExpr() (string, error) { return sym.Raw.Symbol() }
