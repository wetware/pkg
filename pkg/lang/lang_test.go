package lang_test

import (
	"reflect"
	"testing"

	"github.com/spy16/parens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ww "github.com/wetware/ww/pkg"
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
		val:  mustInt(100),
		want: "100",
	}, {
		val:  mustInt(-100),
		want: "-100",
	}, {
		val:  mustFloat(0.123456),
		want: "0.123456",
	}, {
		val:  mustFloat(-0.123456),
		want: "-0.123456",
	}, {
		val:  mustFloat(0.12345678),
		want: "0.12345678",
	}, {
		val:  mustFrac(1, 2),
		want: "1/2",
	}, {
		val:  mustFrac(-1, 2),
		want: "-1/2",
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
			sexpr, err := tt.val.(parens.SExpressable).SExpr()
			require.NoError(t, err)

			assert.Equal(t, tt.want, sexpr)
		})
	}
}

func TestConj(t *testing.T) {
	for _, tt := range []struct {
		desc      string
		col, want ww.Any
		vs        []ww.Any
		wantErr   bool
	}{
		// list
		{
			desc: "(conj () 0 1 2 3)",
			col:  mustList(),
			vs:   []ww.Any{mustInt(0), mustInt(1), mustInt(2), mustInt(3)},
			want: mustList(mustInt(3), mustInt(2), mustInt(1), mustInt(0)),
		},
		{
			desc: "(conj (0) 1 2 3)",
			col:  mustList(mustInt(0)),
			vs:   []ww.Any{mustInt(1), mustInt(2), mustInt(3)},
			want: mustList(mustInt(3), mustInt(2), mustInt(1), mustInt(0)),
		},
		// vector
		{
			desc: "(conj [] 0 1 2 3)",
			col:  mustVector(),
			vs:   []ww.Any{mustInt(0), mustInt(1), mustInt(2), mustInt(3)},
			want: mustVector(mustInt(0), mustInt(1), mustInt(2), mustInt(3)),
		},
		{
			desc: "(conj [0] 1 2 3)",
			col:  mustVector(mustInt(0)),
			vs:   []ww.Any{mustInt(1), mustInt(2), mustInt(3)},
			want: mustVector(mustInt(0), mustInt(1), mustInt(2), mustInt(3)),
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := lang.Conj(tt.col, mustList(tt.vs...))
			if tt.wantErr {
				assert.Error(t, err)
			} else if assert.NoError(t, err) {
				assert.Equal(t, mustSExpr(tt.want), mustSexpr(got))
			}
		})
	}
}

func mustSexpr(any interface{}) string {
	s, err := any.(parens.SExpressable).SExpr()
	if err != nil {
		panic(err)
	}
	return s
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

func mustList(vs ...ww.Any) lang.List {
	l, err := lang.NewList(capnp.SingleSegment(nil), vs...)
	if err != nil {
		panic(err)
	}

	return l
}
