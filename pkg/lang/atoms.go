package lang

import (
	"fmt"

	"github.com/wetware/ww/internal/api"
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

// SExpr returns a valid s-expression for nil.
func (Nil) SExpr() (string, error) { return "nil", nil }

// Data for the Nil type
func (Nil) Data() mem.Value { return mem.NilValue }

// Bool represents a boolean value.
type Bool struct{ mem.Value }

// NewBool .
func NewBool(a capnp.Arena, b bool) (bl Bool, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if bl.Raw, err = api.NewRootValue(seg); err == nil {
		bl.Raw.SetBool(b)
	}

	return
}

// SExpr returns a valid s-expression representing Bool.
func (b Bool) SExpr() (string, error) { return b.String(), nil }

func (b Bool) String() string {
	if b.Raw.Bool() {
		return "true"
	}
	return "false"
}

// Char represents a character literal.  For example, \a, \b, \1, \∂ etc are
// valid character literals. In addition, special literals like \newline, \space
// etc are supported by the reader.
type Char struct{ mem.Value }

// NewChar .
func NewChar(a capnp.Arena, r rune) (c Char, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if c.Raw, err = api.NewRootValue(seg); err == nil {
		c.Raw.SetChar(r)
	}

	return
}

// SExpr returns a valid s-expression representing Char.
func (c Char) SExpr() (string, error) {
	return fmt.Sprintf("\\%c", c.Raw.Char()), nil
}

// String represents double-quoted string literals. String Form represents
// the true string value obtained from the reader. Escape sequences are not
// applicable at this level.
type String struct{ mem.Value }

// NewString .
func NewString(a capnp.Arena, s string) (str String, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if str.Raw, err = api.NewRootValue(seg); err == nil {
		err = str.Raw.SetStr(s)
	}

	return
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
func NewKeyword(a capnp.Arena, s string) (kw Keyword, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if kw.Raw, err = api.NewRootValue(seg); err == nil {
		err = kw.Raw.SetKeyword(s)
	}

	return
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
func NewSymbol(a capnp.Arena, s string) (sym Symbol, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if sym.Raw, err = api.NewRootValue(seg); err == nil {
		err = sym.Raw.SetSymbol(s)
	}

	return
}

// SExpr returns a valid s-expression for the symbol
func (s Symbol) SExpr() (string, error) {
	sym, err := s.Raw.Symbol()
	if err != nil {
		return "", err
	}

	return sym, nil
}
