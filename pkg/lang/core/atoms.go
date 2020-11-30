package core

import (
	"fmt"

	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/mem"
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

// MemVal returns the memory value.
func (Nil) MemVal() mem.Value { return mem.NilValue }

// Bool represents a boolean type.
type Bool interface {
	ww.Any
	Bool() bool
}

// NewBool using the built-in implementation.
func NewBool(a capnp.Arena, b bool) (Bool, error) {
	val, err := mem.NewValue(a)
	if err == nil {
		val.Raw.SetBool(b)
	}

	return boolValue{val}, err
}

type boolValue struct{ mem.Value }

func (b boolValue) Bool() bool { return b.Raw.Bool() }

func (b boolValue) String() string {
	if b.Bool() {
		return "true"
	}
	return "false"
}

// Char represents a character literal.  For example, \a, \b, \1, \∂ etc are
// valid character literals. In addition, special literals like \newline, \space
// etc are supported by the reader.
type Char interface {
	ww.Any
	Char() rune
}

type charValue struct{ mem.Value }

// NewChar using the built-in implementation.
func NewChar(a capnp.Arena, r rune) (Char, error) {
	val, err := mem.NewValue(a)
	if err == nil {
		val.Raw.SetChar(r)
	}

	return charValue{val}, err
}

func (c charValue) Char() rune { return c.Raw.Char() }

func (c charValue) String() string { return fmt.Sprintf("\\%c", c.Char()) }

// String represents text. Escape sequences are not applicable at this level.
type String interface {
	ww.Any
	String() (string, error)
}

type stringValue struct{ mem.Value }

// NewString using the built-in implementation
func NewString(a capnp.Arena, s string) (String, error) {
	val, err := mem.NewValue(a)
	if err == nil {
		err = val.Raw.SetStr(s)
	}

	return stringValue{val}, err
}

func (str stringValue) String() (s string, err error) {
	if s, err = str.Raw.Str(); err == nil {
		s = "\"" + s + "\""
	}

	return
}

// Render the string into a parseable s-expression.
func (str stringValue) Render() (string, error) { return str.String() }

func (str stringValue) Count() (int, error) {
	s, err := str.Raw.Str()
	return len(s), err
}

// Keyword represents a keyword literal.
type Keyword interface {
	ww.Any
	Keyword() (string, error)
}

type keywordValue struct{ mem.Value }

// NewKeyword using the built-in implementation
func NewKeyword(a capnp.Arena, s string) (Keyword, error) {
	val, err := mem.NewValue(a)
	if err == nil {
		err = val.Raw.SetKeyword(s)
	}

	return keywordValue{val}, err
}

func (kw keywordValue) Keyword() (string, error) { return kw.Raw.Keyword() }

func (kw keywordValue) Render() (string, error) {
	s, err := kw.Keyword()
	if err != nil {
		return "", err
	}

	return ":" + s, nil
}

// Symbol represents a name given to a value in memory.
type Symbol interface {
	ww.Any
	Symbol() (string, error)
}

type symbolValue struct{ mem.Value }

// NewSymbol .
func NewSymbol(a capnp.Arena, s string) (Symbol, error) {
	val, err := mem.NewValue(a)
	if err == nil {
		err = val.Raw.SetSymbol(s)
	}

	return symbolValue{val}, err
}

func (sym symbolValue) Symbol() (string, error) { return sym.Raw.Symbol() }

func (sym symbolValue) Render() (string, error) { return sym.Symbol() }
