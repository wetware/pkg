package mem

import (
	"github.com/wetware/ww/internal/api"
	capnp "zombiezen.com/go/capnproto2"
)

// NilValue is a singleton with a precomputed nil value.
var NilValue api.Value

// Value is a persistent datastructure in memory.
type Value api.Value

// NewValue allocates a new value in the supplied arena.
func NewValue(a capnp.Arena) (Value, error) {
	_, seg, err := capnp.NewMessage(a)
	if err != nil {
		return Value{}, err
	}

	val, err := api.NewRootValue(seg)
	return Value(val), err
}

// Bytes represents the raw data underlying the Value.
func (v Value) Bytes() []byte { return api.Value(v).Segment().Data() }

// MemVal returns the underlying value.  It is provided as
// a convenience for the `lang` package, which embeds Value
// into all datatypes.
func (v Value) MemVal() api.Value { return api.Value(v) }

// IsNil returns true if the Value is nil.
func IsNil(v api.Value) bool { return api.Value(v).Which() == api.Value_Which_nil }

// Bytes returns the non-canonical byte array that underpins
// the value.
func Bytes(v api.Value) []byte { return v.Segment().Data() }
