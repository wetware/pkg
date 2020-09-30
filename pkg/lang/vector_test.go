package lang_test

import (
	"fmt"
	"testing"

	"github.com/spy16/parens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang"
	capnp "zombiezen.com/go/capnproto2"
)

func TestNewVector(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		desc string
		vs   []ww.Any
	}{{
		desc: "empty",
		vs:   []ww.Any{},
	}, {
		desc: "single",
		vs:   []ww.Any{mustKeyword("specimen")},
	}, {
		desc: "multi",
		vs: []ww.Any{
			mustKeyword("keyword"),
			mustString("string"),
			mustSymbol("symbol"),
			mustChar('🧠')},
	}, {
		desc: "multinode",
		vs:   valueRange(64), // overflow single node
	}, {
		desc: "multibranch",
		vs:   valueRange(1025), // tree w/ single branch-node => max size of 1024
	}} {
		t.Run(tt.desc, func(t *testing.T) {
			vec, err := lang.NewVector(capnp.SingleSegment(nil), tt.vs...)
			if !assert.NoError(t, err) {
				return
			}

			for i, want := range tt.vs {
				got, err := vec.EntryAt(i)
				if !assert.NoError(t, err) {
					break
				}

				assert.Equal(t, mustSExpr(want), mustSExpr(got),
					"expected %s, got %s", mustSExpr(want), mustSExpr(got))
			}
		})
	}
}

func TestVectorStringer(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		desc, want string
		vec        lang.Vector
	}{{
		desc: "empty",
		vec:  mustVector(),
		want: "[]",
	}, {
		desc: "single",
		vec:  mustVector(mustKeyword("specimen")),
		want: "[:specimen]",
	}, {
		desc: "multi",
		vec: mustVector(
			mustKeyword("keyword"),
			mustString("string"),
			mustSymbol("symbol"),
			mustChar('🧠')),
		want: "[:keyword \"string\" symbol \\🧠]",
	}} {
		t.Run(tt.desc, func(t *testing.T) {
			assert.Equal(t, tt.want, mustSExpr(tt.vec),
				"expected %s, got %s", tt.want, mustSExpr(tt.vec))
		})
	}
}

func TestAssoc(t *testing.T) {
	t.Parallel()

	t.Run("Append", func(t *testing.T) {
		for _, tt := range []struct {
			desc, want string
			vec        lang.Vector
			add        parens.Any
		}{{
			desc: "empty",
			vec:  mustVector(),
			add:  mustKeyword("keyword"),
			want: "[:keyword]",
		}, {
			desc: "non-empty",
			vec: mustVector(
				mustKeyword("keyword"),
				mustString("string"),
				mustSymbol("symbol"),
				mustChar('🧠')),
			add:  mustKeyword("added"),
			want: "[:keyword \"string\" symbol \\🧠 :added]",
		}} {
			t.Run(tt.desc, func(t *testing.T) {
				orig := mustSExpr(tt.vec)
				defer func() {
					require.Equal(t, orig, mustSExpr(tt.vec),
						"IMMUTABILITY VIOLATION")
				}()

				cnt, err := tt.vec.Count()
				require.NoError(t, err)

				got, err := tt.vec.Assoc(cnt, tt.add)
				if assert.NoError(t, err) {
					assert.Equal(t, tt.want, mustSExpr(got),
						"expected %s, got %s", tt.want, mustSExpr(tt.vec))
				}
			})
		}
	})

	t.Run("Update", func(t *testing.T) {
		for _, tt := range []struct {
			want string
			vec  lang.Vector
			add  parens.Any
			idx  int
		}{{
			vec: mustVector(
				mustKeyword("keyword"),
				mustString("string"),
				mustSymbol("symbol"),
				mustChar('🧠')),
			add:  mustKeyword("added"),
			want: "[:added \"string\" symbol \\🧠]",
			idx:  0,
		}, {
			vec: mustVector(
				mustKeyword("keyword"),
				mustString("string"),
				mustSymbol("symbol"),
				mustChar('🧠')),
			add:  mustKeyword("added"),
			want: "[:keyword :added symbol \\🧠]",
			idx:  1,
		}, {
			vec: mustVector(
				mustKeyword("keyword"),
				mustString("string"),
				mustSymbol("symbol"),
				mustChar('🧠')),
			add:  mustKeyword("added"),
			want: "[:keyword \"string\" symbol :added]",
			idx:  3,
		}} {
			t.Run(fmt.Sprintf("%d", tt.idx), func(t *testing.T) {
				orig := mustSExpr(tt.vec)
				defer func() {
					require.Equal(t, orig, mustSExpr(tt.vec),
						"IMMUTABILITY VIOLATION")
				}()

				got, err := tt.vec.Assoc(tt.idx, tt.add)
				if assert.NoError(t, err) {
					assert.Equal(t, tt.want, mustSExpr(got),
						"expected %s, got %s", tt.want, mustSExpr(got))
				}
			})
		}
	})
}

func TestVectorPop(t *testing.T) {
	for _, tt := range []struct {
		desc      string
		vec, want lang.Vector
		wantErr   bool
	}{{
		desc:    "empty",
		vec:     mustVector(),
		wantErr: true,
	}, {
		desc: "single",
		vec:  mustVector(mustKeyword("keyword")),
		want: mustVector(),
	}, {
		desc: "multi",
		vec: mustVector(
			mustKeyword("keyword"),
			mustString("string"),
			mustSymbol("symbol"),
			mustChar('🧠')),
		want: mustVector(
			mustKeyword("keyword"),
			mustString("string"),
			mustSymbol("symbol")),
	}, {
		desc: "multinode",
		vec:  mustVector(valueRange(64)...), // overflow single node
		want: mustVector(valueRange(63)...),
	}, {
		desc: "multibranch",
		vec:  mustVector(valueRange(1025)...), // tree w/ single branch-node => max size of 1024
		want: mustVector(valueRange(1024)...),
	}} {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := tt.vec.Pop()
			if err != nil && tt.wantErr {
				assert.Error(t, err, "expected error, got nil")
				return
			}

			assertVectEq(t, tt.want, got)
		})
	}
}

func TestVectorEquality(t *testing.T) {
	for _, tt := range []struct {
		desc      string
		v         lang.Vector
		newVector func() lang.Vector
	}{
		{
			desc: "basic",
			v:    mustVector(mustInt(0), mustInt(1), mustInt(2)),
			newVector: func() lang.Vector {
				return mustVector(mustInt(0), mustInt(1), mustInt(2))
			},
		},
		{
			desc: "pop from tail",
			v:    mustVector(mustInt(0), mustInt(1), mustInt(2)),
			newVector: func() lang.Vector {
				v := mustVector(mustInt(0), mustInt(1), mustInt(2), mustInt(3))
				any, err := lang.Pop(v)
				require.NoError(t, err)
				return any.(lang.Vector)
			},
		},
		{
			desc: "pop from leaf",
			v:    vectorRange(32),
			newVector: func() lang.Vector {
				v := vectorRange(33)
				any, err := lang.Pop(v)
				require.NoError(t, err)
				return any.(lang.Vector)
			},
		},
		{
			desc: "pop multiple across tail boundary",
			v:    vectorRange(30),
			newVector: func() lang.Vector {
				var err error
				var any ww.Any = vectorRange(35)
				for i := 0; i < 5; i++ {
					if any, err = lang.Pop(any); err != nil {
						panic(err)
					}
				}

				return any.(lang.Vector)
			},
		},
		{
			desc: "complex",
			v: mustVector(
				mustKeyword("keyword"),
				mustFrac(1, 32),
				mustVector(mustFloat(3.14), mustString("string")),
			),
			newVector: func() lang.Vector {
				var err error
				v := mustVector(mustKeyword("keyword"), mustFrac(1, 32))
				v, err = v.Assoc(2, mustVector(mustFloat(3.14), mustString("string")))
				if err != nil {
					panic(err)
				}
				return v
			},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			assert.True(t, lang.Eq(tt.v, tt.newVector()),
				"expected %s, got %s", mustSexpr(tt.v), mustSexpr(tt.newVector()))
		})
	}
}

func mustVector(vs ...ww.Any) lang.Vector {
	vec, err := lang.NewVector(capnp.SingleSegment(nil), vs...)
	if err != nil {
		panic(err)
	}

	return vec
}

func vectorRange(n int) lang.Vector {
	v, err := lang.NewVector(capnp.SingleSegment(nil), valueRange(n)...)
	if err != nil {
		panic(err)
	}

	return v
}

func valueRange(n int) []ww.Any {
	vs := make([]ww.Any, n)
	for i := 0; i < n; i++ {
		vs[i] = mustKeyword(fmt.Sprintf("%d", i))
	}
	return vs
}

func assertVectEq(t *testing.T, want, got lang.Vector) (ok bool) {
	wantcnt, err := want.Count()
	assert.NoError(t, err)

	gotcnt, err := got.Count()
	assert.NoError(t, err)

	if wantcnt != gotcnt {
		t.Errorf("want len=%d, got len=%d", wantcnt, gotcnt)
		return
	}

	for i := 0; i < wantcnt; i++ {
		w, err := want.EntryAt(i)
		if !assert.NoError(t, err) {
			return
		}

		g, err := got.EntryAt(i)
		if !assert.NoError(t, err) {
			return
		}

		if !assert.Equal(t, mustSExpr(w), mustSExpr(g)) {
			return
		}
	}

	return true
}

func mustSExpr(v parens.Any) string {
	sexpr, err := v.(parens.SExpressable).SExpr()
	if err != nil {
		panic(err)
	}

	return sexpr
}
