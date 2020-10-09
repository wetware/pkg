package lang

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/mem"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	_ Numerical = (*Int64)(nil)
	_ Numerical = (*Float64)(nil)
	_ Numerical = (*BigInt)(nil)
	_ Numerical = (*BigFloat)(nil)
	_ Numerical = (*Frac)(nil)

	unit big.Int
)

func init() {
	unit.SetInt64(1)
}

// Numerical value
type Numerical interface {
	ww.Any
	SymbolProvider
	Comparable
}

// Int64 represents a 64-bit signed integer.
type Int64 struct{ mem.Value }

// NewInt64 .
func NewInt64(a capnp.Arena, i int64) (i64 Int64, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if i64.Raw, err = api.NewRootValue(seg); err == nil {
		i64.Raw.SetI64(i)
	}

	return
}

func (i64 Int64) String() string {
	return fmt.Sprintf("%d", i64.Raw.I64())
}

// SExpr returns a valid s-expression for Int64
func (i64 Int64) SExpr() (string, error) { return i64.String(), nil }

// Comp returns 0 if the v == other, -1 if v < other, and 1 if v > other.
func (i64 Int64) Comp(other ww.Any) (int, error) {
	switch o := other.MemVal(); o.Type() {
	case api.Value_Which_i64:
		return compI64(i64.Raw.I64(), o.Raw.I64()), nil

	case api.Value_Which_f64:
		var f big.Float
		return f.SetInt64(i64.Raw.I64()).Cmp(big.NewFloat(o.Raw.F64())), nil

	case api.Value_Which_bigInt:
		return big.NewInt(i64.Raw.I64()).Cmp(other.(BigInt).i), nil

	case api.Value_Which_bigFloat:
		var f big.Float
		return f.SetInt64(i64.Raw.I64()).Cmp(other.(BigFloat).f), nil

	case api.Value_Which_frac:
		var r big.Rat
		return r.SetInt64(i64.Raw.I64()).Cmp(other.(Frac).r), nil

	default:
		return 0, ErrIncomparableTypes

	}
}

// BigInt represents an arbitrary-length signed integer
type BigInt struct {
	i *big.Int
	mem.Value
}

// NewBigInt .
func NewBigInt(a capnp.Arena, i *big.Int) (bi BigInt, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if bi.Raw, err = api.NewRootValue(seg); err == nil {
		err = bi.Raw.SetBigInt(i.Bytes())
	}

	bi.i = i
	return
}

func asBigInt(v mem.Value) (bi BigInt, err error) {
	bi.i = &big.Int{}
	bi.Value = v

	var buf []byte
	if buf, err = v.Raw.BigInt(); err == nil {
		bi.i.SetBytes(buf)
	}

	return
}

func (bi BigInt) String() string { return bi.i.String() }

// SExpr returns a valid s-expression for BigInt
func (bi BigInt) SExpr() (string, error) { return bi.String(), nil }

// Comp returns 0 if the v == other, -1 if v < other, and 1 if v > other.
func (bi BigInt) Comp(other ww.Any) (int, error) {
	switch o := other.MemVal(); o.Type() {
	case api.Value_Which_i64:
		return bi.i.Cmp(big.NewInt(o.Raw.I64())), nil

	case api.Value_Which_f64:
		var f big.Float
		return f.SetInt(bi.i).Cmp(big.NewFloat(o.Raw.F64())), nil

	case api.Value_Which_bigInt:
		return bi.i.Cmp(other.(BigInt).i), nil

	case api.Value_Which_bigFloat:
		var f big.Float
		return f.SetInt(bi.i).Cmp(other.(BigFloat).f), nil

	case api.Value_Which_frac:
		var r big.Rat
		return r.SetFrac(bi.i, &unit).Cmp(other.(Frac).r), nil

	default:
		return 0, ErrIncomparableTypes
	}
}

// Float64 represents a 64-bit floating-point number
type Float64 struct{ mem.Value }

// NewFloat64 .
func NewFloat64(a capnp.Arena, f float64) (f64 Float64, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if f64.Raw, err = api.NewRootValue(seg); err == nil {
		f64.Raw.SetF64(f)
	}

	return
}

func (f64 Float64) String() string {
	return strconv.FormatFloat(f64.Raw.F64(), 'g', -1, 64)
}

// SExpr returns a valid s-expression for Float64
func (f64 Float64) SExpr() (string, error) { return f64.String(), nil }

// Comp returns 0 if the v == other, -1 if v < other, and 1 if v > other.
func (f64 Float64) Comp(other ww.Any) (int, error) {
	switch o := other.MemVal(); o.Type() {
	case api.Value_Which_i64:
		var f big.Float
		return big.NewFloat(f64.Raw.F64()).Cmp(f.SetInt64(o.Raw.I64())), nil

	case api.Value_Which_f64:
		return compF64(f64.Raw.F64(), o.Raw.F64()), nil

	case api.Value_Which_bigInt:
		var f big.Float
		return big.NewFloat(f64.Raw.F64()).Cmp(f.SetInt(other.(BigInt).i)), nil

	case api.Value_Which_bigFloat:
		var bi big.Float
		bi.SetFloat64(f64.Raw.F64())
		return bi.Cmp(other.(BigFloat).f), nil

	case api.Value_Which_frac:
		var r big.Rat
		r.SetFloat64(f64.Raw.F64())
		return r.Cmp(other.(Frac).r), nil

	default:
		return 0, ErrIncomparableTypes

	}
}

// BigFloat represents an arbitrary-precision floating-point number.
type BigFloat struct {
	f *big.Float
	mem.Value
}

// NewBigFloat .
func NewBigFloat(a capnp.Arena, f *big.Float) (bf BigFloat, err error) {
	bf.f = f

	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if bf.Raw, err = api.NewRootValue(seg); err == nil {
		err = bf.Raw.SetBigFloat(f.Text('g', -1))
	}

	return
}

func asBigFloat(v mem.Value) (bf BigFloat, err error) {
	bf.f = &big.Float{}
	bf.Value = v

	var s string
	if s, err = v.Raw.BigFloat(); err == nil {
		if _, ok := bf.f.SetString(s); !ok {
			err = fmt.Errorf("invalid bigfloat format '%s'", s)
		}
	}

	return
}

func (bf BigFloat) String() string { return bf.f.Text('g', -1) }

// SExpr returns a valid s-expression for Float64
func (bf BigFloat) SExpr() (string, error) { return bf.String(), nil }

// Comp returns 0 if the v == other, -1 if v < other, and 1 if v > other.
func (bf BigFloat) Comp(other ww.Any) (int, error) {
	switch o := other.MemVal(); o.Type() {
	case api.Value_Which_i64:
		var f big.Float
		return bf.f.Cmp(f.SetInt64(o.Raw.I64())), nil
	case api.Value_Which_f64:
		return bf.f.Cmp(big.NewFloat(o.Raw.F64())), nil
	case api.Value_Which_bigInt:
		var f big.Float
		return bf.f.Cmp(f.SetInt(other.(BigInt).i)), nil

	case api.Value_Which_bigFloat:
		return bf.f.Cmp(other.(BigFloat).f), nil

	case api.Value_Which_frac:
		var r big.Rat
		bf.f.Rat(&r)
		return r.Cmp(other.(Frac).r), nil

	default:
		return 0, ErrIncomparableTypes

	}
}

// Frac represents a rational number a/b of arbitrary precision.
type Frac struct {
	r *big.Rat
	mem.Value
}

// NewFrac .
func NewFrac(a capnp.Arena, r *big.Rat) (f Frac, err error) {
	f.r = r

	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if f.Raw, err = api.NewRootValue(seg); err != nil {
		return
	}

	var frac api.Frac
	if frac, err = f.Raw.NewFrac(); err != nil {
		return
	}

	if err = frac.SetNumer(r.Num().Bytes()); err != nil {
		return
	}

	if err = frac.SetDenom(r.Denom().Bytes()); err != nil {
		return
	}

	return
}

func asFrac(v mem.Value) (f Frac, err error) {
	f.r = &big.Rat{}
	f.Value = v

	var fv api.Frac
	if fv, err = v.Raw.Frac(); err != nil {
		return
	}

	var nbuf, dbuf []byte
	if nbuf, err = fv.Numer(); err != nil {
		return
	}

	if dbuf, err = fv.Denom(); err != nil {
		return
	}

	var numer, denom big.Int
	numer.SetBytes(nbuf)
	denom.SetBytes(dbuf)

	f.r.SetFrac(&numer, &denom)
	return
}

func (f Frac) String() string { return f.r.String() }

// SExpr returns a valid s-expression for Frac
func (f Frac) SExpr() (string, error) { return f.String(), nil }

// Comp returns true if the other value is numerical and has the same value.
func (f Frac) Comp(other ww.Any) (int, error) {
	switch o := other.MemVal(); o.Type() {
	case api.Value_Which_i64:
		var r big.Rat
		return f.r.Cmp(r.SetFrac(big.NewInt(o.Raw.I64()), &unit)), nil

	case api.Value_Which_f64:
		var r big.Rat
		return f.r.Cmp(r.SetFloat64(o.Raw.F64())), nil

	case api.Value_Which_bigInt:
		var r big.Rat
		return f.r.Cmp(r.SetFrac(other.(BigInt).i, &unit)), nil

	case api.Value_Which_bigFloat:
		var r big.Rat
		other.(BigFloat).f.Rat(&r)
		return f.r.Cmp(&r), nil

	case api.Value_Which_frac:
		return f.r.Cmp(other.(Frac).r), nil

	default:
		return 0, ErrIncomparableTypes

	}
}

func compI64(a, b int64) int {
	switch {
	case a == b:
		return 0
	case a > b:
		return 1
	default:
		return -1
	}
}

func compF64(a, b float64) int {
	switch {
	case a == b:
		return 0
	case a > b:
		return 1
	default:
		return -1
	}
}
