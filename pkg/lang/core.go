package lang

import (
	"github.com/wetware/ww/internal/api"
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

type apiValueProvider interface {
	Value() api.Value
}
