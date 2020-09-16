package lang_test

import (
	"reflect"
	"testing"

	"github.com/spy16/parens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/pkg/lang"
	capnp "zombiezen.com/go/capnproto2"
)

func TestSExpr(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		val  parens.Any
		want string
	}{{
		val:  lang.Nil{},
		want: "nil",
	}, {
		val:  lang.True,
		want: "true",
	}, {
		val:  lang.False,
		want: "false",
	}, {
		val:  parens.Int64(100), // TODO(xxx): replace
		want: "100",
	}, {
		val:  parens.Int64(-100), // TODO(xxx): replace
		want: "-100",
	}, {
		val:  parens.Float64(0.123456), // TODO(xxx): replace
		want: "0.123456",
	}, {
		val:  parens.Float64(-0.123456), // TODO(xxx): replace
		want: "-0.123456",
	}, {
		val:  parens.Float64(0.12345678), // TODO(xxx): replace
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
	}, {
		val:  mustList(mustString("hello"), mustString("world")),
		want: `("hello" "world")`,
	}} {
		t.Run(reflect.TypeOf(tt.val).String(), func(t *testing.T) {
			sexpr, err := tt.val.SExpr()
			require.NoError(t, err)

			assert.Equal(t, tt.want, sexpr)
		})
	}
}

func mustSymbol(s string) lang.Symbol {
	sym, err := lang.NewSymbol(capnp.SingleSegment(nil), s)
	if err != nil {
		panic(err)
	}

	return sym
}

func mustKeyword(s string) lang.Keyword {
	kw, err := lang.NewKeyword(capnp.SingleSegment(nil), s)
	if err != nil {
		panic(err)
	}

	return kw
}

func mustString(s string) lang.String {
	str, err := lang.NewString(capnp.SingleSegment(nil), s)
	if err != nil {
		panic(err)
	}

	return str
}

func mustChar(r rune) lang.Char {
	c, err := lang.NewChar(capnp.SingleSegment(nil), r)
	if err != nil {
		panic(err)
	}

	return c
}

func mustList(vs ...parens.Any) lang.List {
	l, err := lang.NewList(capnp.SingleSegment(nil), vs...)
	if err != nil {
		panic(err)
	}

	return l
}

// import (
// 	"strings"
// 	"testing"

// 	"github.com/spy16/sabre/runtime"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// 	"github.com/wetware/ww/pkg/lang"
// )

// func TestBind(t *testing.T) {
// 	ww := lang.New(nil)

// 	t.Run("BindDoc", func(t *testing.T) {
// 		for _, tC := range []struct {
// 			desc, symbol string
// 			val          runtime.Value
// 			doc          []string
// 		}{{
// 			desc: "basic",
// 			val:  runtime.String("foo"),
// 			doc:  []string{"foo", "bar"},
// 		}, {
// 			desc: "whitespace",
// 			val:  runtime.String("foo"),
// 			doc:  []string{"foo", "bar", "\n", "\n"},
// 		}} {
// 			t.Run(tC.desc, func(t *testing.T) {
// 				require.NoError(t, ww.BindDoc(tC.symbol, tC.val, tC.doc...))

// 				v, err := ww.Resolve(tC.symbol)
// 				assert.NoError(t, err)
// 				assert.Equal(t, tC.val.String(), v.String())

// 				assert.Equal(t, doc(tC.doc), ww.Doc(tC.symbol))
// 			})
// 		}
// 	})

// 	t.Run("Bind", func(t *testing.T) {
// 		require.NoError(t, ww.Bind("foo", runtime.String("foo")), runtime.String("foo"))

// 		v, err := ww.Resolve("foo")
// 		assert.NoError(t, err)
// 		assert.Equal(t, runtime.String("foo").String(), v.String())
// 		assert.Equal(t, "", ww.Doc("foo"))
// 	})

// 	t.Run("ResolveMissing", func(t *testing.T) {
// 		_, err := ww.Resolve("fail")
// 		assert.EqualError(t, err, runtime.ErrNotFound.Error())
// 	})
// }

// func doc(ss []string) string {
// 	return strings.TrimSpace(strings.Join(ss, "\n"))
// }
