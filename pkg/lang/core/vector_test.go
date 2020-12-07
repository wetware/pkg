package core_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/core"
	capnp "zombiezen.com/go/capnproto2"
)

func TestEmptyVector(t *testing.T) {
	t.Parallel()

	require.NotZero(t, core.EmptyVector,
		"zero-value empty vector is invalid (shift is missing)")

	t.Run("NewVector", func(t *testing.T) {
		v, err := core.NewVector(nil)
		assert.NoError(t, err)

		eq, err := core.Eq(core.EmptyVector, v)
		require.NoError(t, err)
		assert.True(t, eq)
	})

	t.Run("Count", func(t *testing.T) {
		cnt, err := core.EmptyVector.Count()
		assert.NoError(t, err)
		assert.Zero(t, cnt)
	})

	t.Run("EntryAt", func(t *testing.T) {
		t.Parallel()

		v, err := core.EmptyVector.EntryAt(0)
		assert.EqualError(t, err, core.ErrIndexOutOfBounds.Error())
		assert.Nil(t, v)
	})

	t.Run("Pop", func(t *testing.T) {
		tail, err := core.EmptyVector.Pop()
		assert.True(t, errors.Is(err, core.ErrIllegalState),
			"'%s' is not ErrIllegalState", err)

		assert.Nil(t, tail)
	})

	t.Run("Conj", func(t *testing.T) {
		v, err := core.EmptyVector.Conj(mustInt(0))
		assert.NoError(t, err)

		v2, err := core.NewVector(nil)
		assert.NoError(t, err)

		v3, err := core.Conj(v2, mustInt(0))
		assert.NoError(t, err)

		eq, err := core.Eq(v, v3)
		require.NoError(t, err)

		assert.True(t, eq, "vector v should be equal to v2.")
	})
}

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
			vec, err := core.NewVector(capnp.SingleSegment(nil), tt.vs...)
			if !assert.NoError(t, err) {
				return
			}

			for i, want := range tt.vs {
				got, err := vec.EntryAt(i)
				if !assert.NoError(t, err) {
					break
				}

				assert.Equal(t, mustRender(want), mustRender(got),
					"expected %s, got %s", mustRender(want), mustRender(got))
			}
		})
	}
}

func TestVectorRender(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		desc, want string
		vec        core.Vector
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
			assert.Equal(t, tt.want, mustRender(tt.vec),
				"expected %s, got %s", tt.want, mustRender(tt.vec))
		})
	}
}

func TestAssoc(t *testing.T) {
	t.Parallel()

	t.Run("Append", func(t *testing.T) {
		for _, tt := range []struct {
			desc, want string
			vec        core.Vector
			add        ww.Any
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
				orig := mustRender(tt.vec)
				defer func() {
					require.Equal(t, orig, mustRender(tt.vec),
						"IMMUTABILITY VIOLATION")
				}()

				cnt, err := tt.vec.Count()
				require.NoError(t, err)

				got, err := tt.vec.Assoc(cnt, tt.add)
				if assert.NoError(t, err) {
					assert.Equal(t, tt.want, mustRender(got),
						"expected %s, got %s", tt.want, mustRender(tt.vec))
				}
			})
		}
	})

	t.Run("Update", func(t *testing.T) {
		for _, tt := range []struct {
			want string
			vec  core.Vector
			add  ww.Any
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
				orig := mustRender(tt.vec)
				defer func() {
					require.Equal(t, orig, mustRender(tt.vec),
						"IMMUTABILITY VIOLATION")
				}()

				got, err := tt.vec.Assoc(tt.idx, tt.add)
				if assert.NoError(t, err) {
					assert.Equal(t, tt.want, mustRender(got),
						"expected %s, got %s", tt.want, mustRender(got))
				}
			})
		}
	})
}

func TestVectorPop(t *testing.T) {
	for _, tt := range []struct {
		desc      string
		vec, want core.Vector
	}{{
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
			assert.NoError(t, err)
			assertVectEq(t, tt.want, got)
		})
	}
}

func TestVectorEquality(t *testing.T) {
	for _, tt := range []struct {
		desc      string
		v         core.Vector
		newVector func() core.Vector
	}{
		{
			desc: "basic",
			v:    mustVector(mustInt(0), mustInt(1), mustInt(2)),
			newVector: func() core.Vector {
				return mustVector(mustInt(0), mustInt(1), mustInt(2))
			},
		},
		{
			desc: "pop from tail",
			v:    mustVector(mustInt(0), mustInt(1), mustInt(2)),
			newVector: func() core.Vector {
				v := mustVector(mustInt(0), mustInt(1), mustInt(2), mustInt(3))
				v, err := v.Pop()
				require.NoError(t, err)
				return v
			},
		},
		{
			desc: "pop from leaf",
			v:    vectorRange(32),
			newVector: func() core.Vector {
				v := vectorRange(33)
				v, err := v.Pop()
				require.NoError(t, err)
				return v
			},
		},
		{
			desc: "pop multiple across tail boundary",
			v:    vectorRange(30),
			newVector: func() core.Vector {
				var err error
				var v = vectorRange(35)
				for i := 0; i < 5; i++ {
					if v, err = v.Pop(); err != nil {
						panic(err)
					}
				}

				return v
			},
		},
		{
			desc: "complex",
			v: mustVector(
				mustKeyword("keyword"),
				mustFrac(1, 32),
				mustVector(mustFloat(3.14), mustString("string")),
			),
			newVector: func() core.Vector {
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
			eq, err := core.Eq(tt.v, tt.newVector())
			require.NoError(t, err)

			assert.True(t, eq,
				"expected %s, got %s", mustRender(tt.v), mustRender(tt.newVector()))
		})
	}
}

func mustVector(vs ...ww.Any) core.Vector {
	vec, err := core.NewVector(capnp.SingleSegment(nil), vs...)
	if err != nil {
		panic(err)
	}

	return vec
}

func vectorRange(n int) core.Vector {
	v, err := core.NewVector(capnp.SingleSegment(nil), valueRange(n)...)
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

func assertVectEq(t *testing.T, want, got core.Vector) (ok bool) {
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

		if !assert.Equal(t, mustRender(w), mustRender(g)) {
			return
		}
	}

	return true
}
