package mem

import "github.com/wetware/ww/internal/api"

// Type of Value
type Type = api.Value_Which

// Value is a persistent datastructure in memory.
type Value struct{ Raw api.Value }

// Type returns the value's type.
func (v Value) Type() Type { return v.Raw.Which() }

// Bytes represents the raw data underlying the Value.
func (v Value) Bytes() []byte { return v.Raw.Segment().Data() }

// Data returns the underlying value.  It is provided as
// a convenience for the `lang` package, which embeds Value
// into all datatypes.
func (v Value) Data() Value { return v }

// Nil returns true if the Value is nil.
func (v Value) Nil() bool { return v.Type() == api.Value_Which_nil }
