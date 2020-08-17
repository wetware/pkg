package core_test

import (
	"fmt"
	"testing"

	"github.com/spy16/sabre/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/pkg/lang/core"
	capnp "zombiezen.com/go/capnproto2"
)

func TestNewVector(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		desc string
		vs   []runtime.Value
	}{{
		desc: "empty",
		vs:   []runtime.Value{},
	}, {
		desc: "single",
		vs:   []runtime.Value{mustKeyword("specimen")},
	}, {
		desc: "multi",
		vs: []runtime.Value{
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

				assert.Equal(t, want.String(), got.String(),
					"expected %s, got %s", want.String(), got.String())
			}
		})
	}
}

func TestVectorStringer(t *testing.T) {
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
			t.Log(tt.vec.String())
			assert.Equal(t, tt.want, tt.vec.String(),
				"expected %s, got %s", tt.want, tt.vec.String())
		})
	}
}

func TestAssoc(t *testing.T) {
	t.Parallel()

	t.Run("Append", func(t *testing.T) {
		for _, tt := range []struct {
			desc, want string
			vec        core.Vector
			add        runtime.Value
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
				orig := tt.vec.String()
				defer func() {
					require.Equal(t, orig, tt.vec.String(),
						"IMMUTABILITY VIOLATION")
				}()

				got, err := tt.vec.Assoc(tt.vec.Count(), tt.add)
				if assert.NoError(t, err) {
					assert.Equal(t, tt.want, got.String(),
						"expected %s, got %s", tt.want, tt.vec.String())
				}
			})
		}
	})

	t.Run("Update", func(t *testing.T) {
		for _, tt := range []struct {
			want string
			vec  core.Vector
			add  runtime.Value
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
				orig := tt.vec.String()
				defer func() {
					require.Equal(t, orig, tt.vec.String(),
						"IMMUTABILITY VIOLATION")
				}()

				got, err := tt.vec.Assoc(tt.idx, tt.add)
				if assert.NoError(t, err) {
					assert.Equal(t, tt.want, got.String(),
						"expected %s, got %s", tt.want, got.String())
				}
			})
		}
	})
}

func TestVectorPop(t *testing.T) {
	for _, tt := range []struct {
		desc      string
		vec, want core.Vector
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

func mustVector(vs ...runtime.Value) core.Vector {
	vec, err := core.NewVector(capnp.SingleSegment(nil), vs...)
	if err != nil {
		panic(err)
	}

	return vec
}

func valueRange(n int) []runtime.Value {
	vs := make([]runtime.Value, n)
	for i := 0; i < n; i++ {
		vs[i] = mustKeyword(fmt.Sprintf("%d", i))
	}
	return vs
}

func assertVectEq(t *testing.T, want, got core.Vector) (ok bool) {
	if want.Count() != got.Count() {
		t.Errorf("want len=%d, got len=%d", want.Count(), got.Count())
		return
	}

	for i := 0; i < want.Count(); i++ {
		wat, err := want.EntryAt(i)
		if !assert.NoError(t, err) {
			return
		}

		gat, err := got.EntryAt(i)
		if !assert.NoError(t, err) {
			fmt.Println(got.Value())
			return
		}

		if !assert.Equal(t, wat.String(), gat.String()) {
			return
		}
	}

	return true
}
