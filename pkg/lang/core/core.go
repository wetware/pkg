// Package core contains built-ins for the wetware language.
package core

import (
	"github.com/spy16/sabre/runtime"
	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	bindutil "github.com/wetware/ww/pkg/lang/util/bind"
)

var (
	// True value of Bool
	True Bool

	//False value of Bool
	False Bool
)

func init() {
	var err error
	if True, err = NewBool(true); err != nil {
		panic(err)
	}

	if False, err = NewBool(false); err != nil {
		panic(err)
	}
}

// Bind root anchor to a namespace registration function.
func Bind(root ww.Anchor) bindutil.BinderFunc {
	return func(r runtime.Runtime) error {
		return bindutil.BindList(r, []bindutil.Binding{
			// logical constants
			bindutil.Bind("nil", Nil{},
				"Represents logical false. Same as false",
			),
			bindutil.Bind("true", True,
				"Represents logical true",
			),
			bindutil.Bind("false", False,
				"Represents logical false",
			),

			// anchor API
			bindutil.Bind("ls", list{root},
				"(ls <path>)",
				"list an anchor path"),
		})
	}
}

// IsNil returns true if the value is logically nil
func IsNil(v runtime.Value) (null bool) {
	if null = v == nil; !null {
		_, null = v.(Nil)
	}

	return
}

type apiValueProvider interface {
	Value() api.Value
}
