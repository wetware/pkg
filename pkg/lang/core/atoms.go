package core

import ww "github.com/wetware/ww/pkg"

// Nil represents a null value
type Nil interface {
	ww.Any
	Nil()
}

// Bool is a boolean value.
type Bool interface {
	ww.Any
	Bool() bool
}

// Symbol represents a name given to a value in memory.
type Symbol interface {
	ww.Any
	Symbol() (string, error)
}

// String represents text. Escape sequences are not applicable at this level.
type String interface {
	ww.Any
	String() (string, error)
}

// Keyword is a self-referencing symbol.
type Keyword interface {
	ww.Any
	Keyword() (string, error)
}

// Char represents a unicode character literal.
type Char interface {
	ww.Any
	Char() rune
}
