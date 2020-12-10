package mem

import (
	"github.com/wetware/ww/internal/api"
	capnp "zombiezen.com/go/capnproto2"
)

// NilValue is a singleton with a precomputed nil value.
var NilValue api.Any

// Value is a persistent datastructure in memory.
type Value api.Any

// NewValue allocates a new value in the supplied arena.
func NewValue(a capnp.Arena) (Value, error) {
	_, seg, err := capnp.NewMessage(a)
	if err != nil {
		return Value{}, err
	}

	val, err := api.NewRootAny(seg)
	return Value(val), err
}

// Bytes represents the raw data underlying the Value.
func (v Value) Bytes() []byte { return api.Any(v).Segment().Data() }

// MemVal returns the underlying value.  It is provided as
// a convenience for the `lang` package, which embeds Value
// into all datatypes.
func (v Value) MemVal() api.Any { return api.Any(v) }

// IsNil returns true if the Value is nil.
func IsNil(v api.Any) bool { return api.Any(v).Which() == api.Any_Which_nil }

// Bytes returns the non-canonical byte array that underpins
// the value.
func Bytes(v api.Any) []byte { return v.Segment().Data() }
