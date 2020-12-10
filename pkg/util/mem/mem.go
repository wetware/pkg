package memutil

import (
	"github.com/wetware/ww/internal/mem"
	capnp "zombiezen.com/go/capnproto2"
)

func Alloc(a capnp.Arena) (mem.Any, error) {
	_, seg, err := capnp.NewMessage(a)
	if err != nil {
		return mem.Any{}, err
	}

	// TODO(performance):  we might not always want to allocate a _root_ value,
	//					   e.g. if the value is to be assigned to a vector index.
	//					   Investigate the implications of root vs non-root and
	//					   consider providing a mechanism for non-root allocation.
	return mem.NewRootAny(seg)
}

// Bytes returns the underlying byte array for the supplied value.
func Bytes(any mem.Any) []byte { return any.Segment().Data() }

// IsNil returns true if the supplied value is nil.
func IsNil(any mem.Any) bool { return any.Which() == mem.Any_Which_nil }
