package mem

import (
	"github.com/wetware/ww/internal/api"
	capnp "zombiezen.com/go/capnproto2"
)

// Type of Value
type Type = api.Value_Which

// Value is a persistent datastructure in memory.
type Value struct{ Raw api.Value }

// NewValue allocates a new value in the supplied arena.
func NewValue(a capnp.Arena) (val Value, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err == nil {
		val.Raw, err = api.NewRootValue(seg)
	}

	return
}

// Type returns the value's type.
func (v Value) Type() Type { return v.Raw.Which() }

// Bytes represents the raw data underlying the Value.
func (v Value) Bytes() []byte { return v.Raw.Segment().Data() }

// MemVal returns the underlying value.  It is provided as
// a convenience for the `lang` package, which embeds Value
// into all datatypes.
func (v Value) MemVal() Value { return v }

// Nil returns true if the Value is nil.
func (v Value) Nil() bool { return v.Type() == api.Value_Which_nil }
