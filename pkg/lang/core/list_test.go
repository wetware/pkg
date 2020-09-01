package core_test

import (
	"testing"

	"github.com/spy16/sabre/runtime"
	"github.com/wetware/ww/pkg/lang/core"
	capnp "zombiezen.com/go/capnproto2"
)

func TestList_Eval(t *testing.T) {
	t.Parallel()

	runEvalTests(t, []evalTestCase{{
		desc: "EmptyList",
		form: core.EmptyList,
		want: core.EmptyList,
	}, {
		desc:    "FirstEvalFailure",
		getEnv:  func() runtime.Runtime { return runtime.New(nil) },
		form:    mustList(mustSymbol("non-existent")),
		wantErr: true,
	}, {
		desc:    "NonInvokable",
		getEnv:  func() runtime.Runtime { return runtime.New(nil) },
		form:    mustList(mustString("not-invokable")),
		wantErr: true,
		// }, {
		// 	desc:   "InvokableNoArgs",
		// 	getEnv: func() runtime.Runtime { return runtime.New(nil) },
		// 	form: mustList(runtime.GoFunc(func(env runtime.Runtime, args ...runtime.Value) (runtime.Value, error) {
		// 		return mustString("called"), nil
		// 	})),
		// 	want:    mustString("called"),
		// 	wantErr: false,
		// }, {
		// 	desc:   "InvokableWithArgs",
		// 	getEnv: func() runtime.Runtime { return runtime.New(nil) },
		// 	form: mustList(runtime.GoFunc(func(env runtime.Runtime, args ...runtime.Value) (runtime.Value, error) {
		// 		return args[0], nil
		// 	}), mustString("hello")),
		// 	want:    mustString("hello"),
		// 	wantErr: false,
	}})
}

func mustList(vs ...runtime.Value) core.List {
	l, err := core.NewList(capnp.SingleSegment(nil), vs...)
	if err != nil {
		panic(err)
	}

	return l
}
