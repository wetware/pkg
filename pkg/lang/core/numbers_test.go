package core_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/core"
	capnp "zombiezen.com/go/capnproto2"
)

func TestInt(t *testing.T) {
	for _, tt := range []struct {
		a, b    ww.Any
		want    int
		wantErr bool
	}{
		{
			a: mustInt(5),
			b: mustFloat(5.0),
		},
		{
			a:    mustInt(5),
			b:    mustFloat(0),
			want: 1,
		},
		{
			a:    mustInt(0),
			b:    mustFloat(3.14),
			want: -1,
		},
		{
			a: mustInt(5),
			b: mustInt(5),
		},
		{
			a:    mustInt(5),
			b:    mustInt(0),
			want: 1,
		},
		{
			a:    mustInt(0),
			b:    mustInt(5),
			want: -1,
		},
		{
			a: mustInt(5),
			b: mustBigFloat(5.0),
		},
		{
			a:    mustInt(5),
			b:    mustBigFloat(0),
			want: 1,
		},
		{
			a:    mustInt(0),
			b:    mustBigFloat(5),
			want: -1,
		},
		{
			a: mustInt(5),
			b: mustBigInt(5),
		},
		{
			a:    mustInt(5),
			b:    mustBigInt(0),
			want: 1,
		},
		{
			a:    mustInt(0),
			b:    mustBigInt(5),
			want: -1,
		},
		{
			a: mustInt(3),
			b: mustFrac(6, 2),
		},
		{
			a:    mustInt(5),
			b:    mustFrac(1, 2),
			want: 1,
		},
		{
			a:    mustInt(0),
			b:    mustFrac(1, 2),
			want: -1,
		},
		{
			a:       mustInt(5),
			b:       core.Nil{},
			wantErr: true,
		},
	} {
		t.Run(compDesc(tt), func(t *testing.T) {
			got, err := tt.a.(core.Comparable).Comp(tt.b)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestFloat(t *testing.T) {
	for _, tt := range []struct {
		a, b    ww.Any
		want    int
		wantErr bool
	}{
		{
			a: mustFloat(3.14),
			b: mustFloat(3.14),
		},
		{
			a:    mustFloat(3.14),
			b:    mustFloat(0),
			want: 1,
		},
		{
			a:    mustFloat(0),
			b:    mustFloat(3.14),
			want: -1,
		},
		{
			a: mustFloat(3.0),
			b: mustInt(3.0),
		},
		{
			a:    mustFloat(3.14),
			b:    mustInt(0),
			want: 1,
		},
		{
			a:    mustFloat(0),
			b:    mustInt(5),
			want: -1,
		},
		{
			a: mustFloat(3.0),
			b: mustBigFloat(3.0),
		},
		{
			a:    mustFloat(3.14),
			b:    mustBigFloat(0),
			want: 1,
		},
		{
			a:    mustFloat(0),
			b:    mustBigFloat(5),
			want: -1,
		},
		{
			a: mustFloat(3.0),
			b: mustBigInt(3),
		},
		{
			a:    mustFloat(3.14),
			b:    mustBigInt(0),
			want: 1,
		},
		{
			a:    mustFloat(0),
			b:    mustBigInt(5),
			want: -1,
		},
		{
			a: mustFloat(3.0),
			b: mustFrac(6, 2),
		},
		{
			a:    mustFloat(3.14),
			b:    mustFrac(1, 2),
			want: 1,
		},
		{
			a:    mustFloat(0),
			b:    mustFrac(1, 2),
			want: -1,
		},
		{
			a:       mustFloat(1.2345),
			b:       core.Nil{},
			wantErr: true,
		},
	} {
		t.Run(compDesc(tt), func(t *testing.T) {
			got, err := tt.a.(core.Comparable).Comp(tt.b)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestBigInt(t *testing.T) {
	for _, tt := range []struct {
		a, b    ww.Any
		want    int
		wantErr bool
	}{
		{
			a: mustBigInt(5),
			b: mustFloat(5.0),
		},
		{
			a:    mustBigInt(5),
			b:    mustFloat(0),
			want: 1,
		},
		{
			a:    mustBigInt(0),
			b:    mustFloat(3.14),
			want: -1,
		},
		{
			a: mustBigInt(5),
			b: mustInt(5),
		},
		{
			a:    mustBigInt(5),
			b:    mustInt(0),
			want: 1,
		},
		{
			a:    mustBigInt(0),
			b:    mustInt(5),
			want: -1,
		},
		{
			a: mustBigInt(5),
			b: mustBigFloat(5.0),
		},
		{
			a:    mustBigInt(5),
			b:    mustBigFloat(0),
			want: 1,
		},
		{
			a:    mustBigInt(0),
			b:    mustBigFloat(5),
			want: -1,
		},
		{
			a: mustBigInt(5),
			b: mustBigInt(5),
		},
		{
			a:    mustBigInt(5),
			b:    mustBigInt(0),
			want: 1,
		},
		{
			a:    mustBigInt(0),
			b:    mustBigInt(5),
			want: -1,
		},
		{
			a: mustBigInt(3),
			b: mustFrac(6, 2),
		},
		{
			a:    mustBigInt(5),
			b:    mustFrac(1, 2),
			want: 1,
		},
		{
			a:    mustBigInt(0),
			b:    mustFrac(1, 2),
			want: -1,
		},
		{
			a:       mustBigInt(5),
			b:       core.Nil{},
			wantErr: true,
		},
	} {
		t.Run(compDesc(tt), func(t *testing.T) {
			got, err := tt.a.(core.Comparable).Comp(tt.b)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestBigFloat(t *testing.T) {
	for _, tt := range []struct {
		a, b    ww.Any
		want    int
		wantErr bool
	}{
		{
			a: mustBigFloat(3.14),
			b: mustFloat(3.14),
		},
		{
			a:    mustBigFloat(3.14),
			b:    mustFloat(0),
			want: 1,
		},
		{
			a:    mustBigFloat(0),
			b:    mustFloat(3.14),
			want: -1,
		},
		{
			a: mustBigFloat(3.0),
			b: mustInt(3.0),
		},
		{
			a:    mustBigFloat(3.14),
			b:    mustInt(0),
			want: 1,
		},
		{
			a:    mustBigFloat(0),
			b:    mustInt(5),
			want: -1,
		},
		{
			a: mustBigFloat(3.0),
			b: mustBigFloat(3.0),
		},
		{
			a:    mustBigFloat(3.14),
			b:    mustBigFloat(0),
			want: 1,
		},
		{
			a:    mustBigFloat(0),
			b:    mustBigFloat(5),
			want: -1,
		},
		{
			a: mustBigFloat(3.0),
			b: mustBigInt(3),
		},
		{
			a:    mustBigFloat(3.14),
			b:    mustBigInt(0),
			want: 1,
		},
		{
			a:    mustBigFloat(0),
			b:    mustBigInt(5),
			want: -1,
		},
		{
			a: mustBigFloat(3.0),
			b: mustFrac(6, 2),
		},
		{
			a:    mustBigFloat(3.14),
			b:    mustFrac(1, 2),
			want: 1,
		},
		{
			a:    mustBigFloat(0),
			b:    mustFrac(1, 2),
			want: -1,
		},
		{
			a:       mustBigFloat(1.2345),
			b:       core.Nil{},
			wantErr: true,
		},
	} {
		t.Run(compDesc(tt), func(t *testing.T) {
			got, err := tt.a.(core.Comparable).Comp(tt.b)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestFrac(t *testing.T) {
	for _, tt := range []struct {
		a, b    ww.Any
		want    int
		wantErr bool
	}{
		{
			a: mustFrac(1, 2),
			b: mustFloat(.5),
		},
		{
			a:    mustFrac(1, 2),
			b:    mustFloat(0),
			want: 1,
		},
		{
			a:    mustFrac(1, 2),
			b:    mustFloat(3.14),
			want: -1,
		},
		{
			a: mustFrac(10, 2),
			b: mustInt(5),
		},
		{
			a:    mustFrac(1, 2),
			b:    mustInt(0),
			want: 1,
		},
		{
			a:    mustFrac(1, 2),
			b:    mustInt(5),
			want: -1,
		},
		{
			a: mustFrac(1, 2),
			b: mustBigFloat(.5),
		},
		{
			a:    mustFrac(1, 2),
			b:    mustBigFloat(0),
			want: 1,
		},
		{
			a:    mustFrac(1, 2),
			b:    mustBigFloat(5.22),
			want: -1,
		},
		{
			a: mustFrac(10, 2),
			b: mustBigInt(5),
		},
		{
			a:    mustFrac(1, 2),
			b:    mustBigInt(0),
			want: 1,
		},
		{
			a:    mustFrac(1, 2),
			b:    mustBigInt(5),
			want: -1,
		},
		{
			a: mustFrac(6, 12),
			b: mustFrac(1, 2),
		},
		{
			a:    mustFrac(10, 10),
			b:    mustFrac(1, 2),
			want: 1,
		},
		{
			a:    mustFrac(1, 10),
			b:    mustFrac(1, 2),
			want: -1,
		},
		{
			a:       mustFrac(1, 2),
			b:       core.Nil{},
			wantErr: true,
		},
	} {
		t.Run(compDesc(tt), func(t *testing.T) {
			got, err := tt.a.(core.Comparable).Comp(tt.b)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func mustFloat(f float64) core.Float64 {
	f64, err := core.NewFloat64(capnp.SingleSegment(nil), f)
	if err != nil {
		panic(err)
	}

	return f64
}

func mustInt(i int64) core.Int64 {
	i64, err := core.NewInt64(capnp.SingleSegment(nil), i)
	if err != nil {
		panic(err)
	}

	return i64
}

func mustFrac(numer, denom int64) core.Fraction {
	f, err := core.NewFraction(capnp.SingleSegment(nil), big.NewRat(numer, denom))
	if err != nil {
		panic(err)
	}

	return f
}

func mustBigFloat(f float64) core.BigFloat {
	bf, err := core.NewBigFloat(capnp.SingleSegment(nil), big.NewFloat(f))
	if err != nil {
		panic(err)
	}

	return bf
}

func mustBigInt(i int64) core.BigInt {
	bi, err := core.NewBigInt(capnp.SingleSegment(nil), big.NewInt(i))
	if err != nil {
		panic(err)
	}

	return bi
}

func compDesc(tt struct {
	a, b    ww.Any
	want    int
	wantErr bool
}) string {
	var sym = "=="
	if tt.want < 0 {
		sym = "<"
	} else if tt.want > 0 {
		sym = ">"
	}

	aname, err := core.Render(tt.a)
	if err != nil {
		panic(err)
	}

	bname, err := core.Render(tt.b)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s%s%s", aname, sym, bname)
}
