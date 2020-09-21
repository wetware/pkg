package mem

import (
	"github.com/wetware/ww/internal/api"
	capnp "zombiezen.com/go/capnproto2"
)

// NilValue is a singleton with a precomputed nil value.
var NilValue Value

func init() {
	_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		panic(err)
	}

	if NilValue.Raw, err = api.NewRootValue(seg); err == nil {
		NilValue.Raw.SetNil()
	}
}
