package core_test

import (
	"reflect"
	"testing"

	"github.com/spy16/sabre/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/wetware/ww/pkg/lang"
	"github.com/wetware/ww/pkg/lang/core"
)

func TestKeywordLookup(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		desc    string
		getEnv  func() runtime.Runtime
		args    []runtime.Value
		want    runtime.Value
		wantErr bool
	}{{
		desc:    "ArityMismatch",
		args:    nil,
		wantErr: true,
	}, {
		desc:    "ArgEvalError",
		args:    []runtime.Value{mustSymbol("test")},
		getEnv:  func() runtime.Runtime { return lang.New(nil) },
		wantErr: true,
	}, {
		desc:   "NotMap",
		args:   []runtime.Value{runtime.Seq(nil)},
		getEnv: func() runtime.Runtime { return lang.New(nil) },
		want:   core.Nil{},
	}, {
		desc: "WithDefault",
		args: []runtime.Value{
			&mockMap{
				Lookup: func(key runtime.Value) runtime.Value {
					return nil
				},
			},
			runtime.Float64(10), // TODO(xxx): replace
		},
		getEnv: func() runtime.Runtime { return lang.New(nil) },
		want:   runtime.Float64(10), // TODO(xxx): replace
	}, {
		desc: "WithoutDefault",
		args: []runtime.Value{
			&mockMap{Lookup: func(key runtime.Value) runtime.Value {
				if runtime.Equals(key, mustKeyword("specimen")) {
					return runtime.Float64(10) // TODO(xxx): replace
				}
				return nil
			}},
			runtime.Float64(10), // TODO(xxx): replace
		},
		getEnv: func() runtime.Runtime { return lang.New(nil) },
		want:   runtime.Float64(10), // TODO(xxx): replace
	}} {
		t.Run(tt.desc, func(t *testing.T) {
			var env runtime.Runtime
			if tt.getEnv != nil {
				env = tt.getEnv()
			}

			got, err := mustKeyword("specimen").Invoke(env, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Keyword.Invoke() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Keyword.Invoke() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_Eval(t *testing.T) {
	t.Parallel()
	runEvalTests(t, []evalTestCase{{
		desc: "Nil",
		form: core.Nil{},
		want: core.Nil{},
	}, {
		desc: "Bool",
		form: core.False,
		want: core.False,
	}, {
		desc: "Float64",
		form: runtime.Float64(0.123456789), // TODO(xxx): replace
		want: runtime.Float64(0.123456789), // TODO(xxx): replace
	}, {
		desc: "Int64",
		form: runtime.Int64(10), // TODO(xxx): replace
		want: runtime.Int64(10), // TODO(xxx): replace
	}, {
		desc: "Char",
		form: mustChar('c'),
		want: mustChar('c'),
	}, {
		desc: "Keyword",
		form: mustKeyword("specimen"),
		want: mustKeyword("specimen"),
	}, {
		desc: "String",
		form: mustString("specimen"),
		want: mustString("specimen"),
	}, {
		desc: "Symbol",
		getEnv: func() runtime.Runtime {
			env := lang.New(nil)
			_ = env.Bind("π", runtime.Float64(3.1412)) // TODO(xxx): replace
			return env
		},
		form: runtime.Symbol{ // TODO(fixme)
			Value:    "π",
			Position: runtime.Position{File: "lisp"},
		},
		want: runtime.Float64(3.1412), // TODO(xxx): replace
	}})
}

func Test_String(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		val  runtime.Value
		want string
	}{{
		val:  core.Nil{},
		want: "nil",
	}, {
		val:  core.True,
		want: "true",
	}, {
		val:  core.False,
		want: "false",
	}, {
		val:  runtime.Int64(100), // TODO(xxx): replace
		want: "100",
	}, {
		val:  runtime.Int64(-100), // TODO(xxx): replace
		want: "-100",
	}, {
		val:  runtime.Float64(0.123456), // TODO(xxx): replace
		want: "0.123456",
	}, {
		val:  runtime.Float64(-0.123456), // TODO(xxx): replace
		want: "-0.123456",
	}, {
		val:  runtime.Float64(0.12345678), // TODO(xxx): replace
		want: "0.123457",
	}, {
		val:  mustChar('π'),
		want: `\π`,
	}, {
		val:  mustKeyword("specimen"),
		want: ":specimen",
	}, {
		val:  mustSymbol("specimen"),
		want: "specimen",
	}, {
		val:  mustString("hello 😎"),
		want: `"hello 😎"`,
	}} {
		t.Run(reflect.TypeOf(tt.val).String(), func(t *testing.T) {
			assert.Equal(t, tt.want, tt.val.String())
		})
	}
}

type mockMap struct {
	runtime.Map

	Lookup func(runtime.Value) runtime.Value
}

func (m *mockMap) Seq() runtime.Seq {
	return runtime.NewSeq()
}

func (m *mockMap) Eval(rt runtime.Runtime) (runtime.Value, error) {
	return m, nil
}

func (m *mockMap) EntryAt(key runtime.Value) runtime.Value {
	return m.Lookup(key)
}

func mustSymbol(s string) core.Symbol {
	sym, err := core.NewSymbol(s)
	if err != nil {
		panic(err)
	}

	return sym
}

func mustKeyword(s string) core.Keyword {
	kw, err := core.NewKeyword(s)
	if err != nil {
		panic(err)
	}

	return kw
}

func mustString(s string) core.String {
	str, err := core.NewString(s)
	if err != nil {
		panic(err)
	}

	return str
}

func mustChar(r rune) core.Char {
	c, err := core.NewChar(r)
	if err != nil {
		panic(err)
	}

	return c
}
