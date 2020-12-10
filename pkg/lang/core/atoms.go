package core

import (
	"fmt"

	"github.com/wetware/ww/internal/mem"
	memutil "github.com/wetware/ww/pkg/util/mem"
	capnp "zombiezen.com/go/capnproto2"
)

var (
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

// Value returns the memory value.
func (Nil) Value() mem.Any { return mem.Any{} }

// Bool represents a boolean type.
type Bool struct{ mem.Any }

// NewBool using the built-in implementation.
func NewBool(a capnp.Arena, b bool) (Bool, error) {
	any, err := memutil.Alloc(a)
	if err == nil {
		any.SetBool(b)
	}

	return Bool{any}, err
}

// Value returns the memory value
func (b Bool) Value() mem.Any { return b.Any }

// Bool returns the boolean value.
func (b Bool) Bool() bool { return b.Value().Bool() }

func (b Bool) String() string {
	if b.Bool() {
		return "true"
	}
	return "false"
}

// Char represents a character literal.  For example, \a, \b, \1, \∂ etc are
// valid character literals. In addition, special literals like \newline, \space
// etc are supported by the reader.
type Char struct{ mem.Any }

// NewChar using the built-in implementation.
func NewChar(a capnp.Arena, r rune) (Char, error) {
	any, err := memutil.Alloc(a)
	if err == nil {
		any.SetChar(r)
	}

	return Char{any}, err
}

// Value returns the memory value
func (c Char) Value() mem.Any { return c.Any }

// Char returns the character as a native rune.
func (c Char) Char() rune { return c.Value().Char() }

func (c Char) String() string { return fmt.Sprintf("\\%c", c.Char()) }

// String represents text. Escape sequences are not applicable at this level.
type String struct{ mem.Any }

// NewString using the built-in implementation
func NewString(a capnp.Arena, s string) (String, error) {
	any, err := memutil.Alloc(a)
	if err == nil {
		err = any.SetStr(s)
	}

	return String{any}, err
}

// Value returns the memory value
func (str String) Value() mem.Any { return str.Any }

func (str String) String() (s string, err error) {
	if s, err = str.Value().Str(); err == nil {
		s = "\"" + s + "\""
	}

	return
}

// Render the string into a parseable s-expression.
func (str String) Render() (string, error) { return str.String() }

// Count returns the number of characters in the string.
func (str String) Count() (int, error) {
	s, err := str.Value().Str()
	return len(s), err
}

// Keyword represents a keyword literal.
type Keyword struct{ mem.Any }

// NewKeyword using the built-in implementation
func NewKeyword(a capnp.Arena, s string) (Keyword, error) {
	any, err := memutil.Alloc(a)
	if err == nil {
		err = any.SetKeyword(s)
	}

	return Keyword{any}, err
}

// Value returns the memory value
func (kw Keyword) Value() mem.Any { return kw.Any }

// Keyword returns the keyword's value as a native string.
func (kw Keyword) Keyword() (string, error) { return kw.Value().Keyword() }

// Render the keyword into a human-readable string.
func (kw Keyword) Render() (string, error) {
	s, err := kw.Keyword()
	if err != nil {
		return "", err
	}

	return ":" + s, nil
}

// Symbol represents a name given to a value in memory.
type Symbol struct{ mem.Any }

// NewSymbol .
func NewSymbol(a capnp.Arena, s string) (Symbol, error) {
	any, err := memutil.Alloc(a)
	if err == nil {
		err = any.SetSymbol(s)
	}

	return Symbol{any}, err
}

// Value returns the memory value
func (sym Symbol) Value() mem.Any { return sym.Any }

// Symbol returns the symbol's value as a native string.
func (sym Symbol) Symbol() (string, error) { return sym.Value().Symbol() }

// Render the symbol into a human-readable string.
func (sym Symbol) Render() (string, error) { return sym.Symbol() }
