package lang

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/spy16/parens"
	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	_ ww.Any = (*Int64)(nil)
	_ ww.Any = (*Float64)(nil)
	_ ww.Any = (*BigInt)(nil)
	_ ww.Any = (*BigFloat)(nil)
	_ ww.Any = (*Frac)(nil)

	_ Comparable = (*Int64)(nil)
	_ Comparable = (*Float64)(nil)
	_ Comparable = (*BigInt)(nil)
	_ Comparable = (*BigFloat)(nil)
	_ Comparable = (*Frac)(nil)

	unit big.Int
)

func init() {
	unit.SetInt64(1)
}

// Int64 represents a 64-bit signed integer.
type Int64 struct {
	v api.Value
}

// NewInt64 .
func NewInt64(a capnp.Arena, i int64) (i64 Int64, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if i64.v, err = api.NewRootValue(seg); err == nil {
		i64.v.SetI64(i)
	}

	return
}

// Value for Int64 type
func (i64 Int64) Value() api.Value {
	return i64.v
}

// SExpr returns a valid s-expression for Int64
func (i64 Int64) SExpr() (string, error) {
	return fmt.Sprintf("%d", i64.v.I64()), nil
}

// Comp returns 0 if the v == other, -1 if v < other, and 1 if v > other.
func (i64 Int64) Comp(other parens.Any) (int, error) {
	switch n := other.(type) {
	case Int64:
		return compI64(i64.v.I64(), n.v.I64()), nil

	case Float64:
		var f big.Float
		return f.SetInt64(i64.v.I64()).Cmp(big.NewFloat(n.v.F64())), nil

	case BigInt:
		return big.NewInt(i64.v.I64()).Cmp(n.i), nil

	case BigFloat:
		var f big.Float
		return f.SetInt64(i64.v.I64()).Cmp(n.f), nil

	case Frac:
		var r big.Rat
		return r.SetInt64(i64.v.I64()).Cmp(n.r), nil

	default:
		return 0, ErrIncomparableTypes

	}
}

// BigInt represents an arbitrary-length signed integer
type BigInt struct {
	i *big.Int
	v api.Value
}

// NewBigInt .
func NewBigInt(a capnp.Arena, i *big.Int) (bi BigInt, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if bi.v, err = api.NewRootValue(seg); err == nil {
		err = bi.v.SetBigInt(i.Bytes())
	}

	bi.i = i
	return
}

func bigIntFromValue(v api.Value) (bi BigInt, err error) {
	bi.i = &big.Int{}
	bi.v = v

	var buf []byte
	if buf, err = v.BigInt(); err == nil {
		bi.i.SetBytes(buf)
	}

	return
}

// Value for BigInt type
func (bi BigInt) Value() api.Value {
	return bi.v
}

// SExpr returns a valid s-expression for BigInt.
func (bi BigInt) SExpr() (string, error) {
	return bi.i.String(), nil
}

// Comp returns 0 if the v == other, -1 if v < other, and 1 if v > other.
func (bi BigInt) Comp(other parens.Any) (int, error) {
	switch n := other.(type) {
	case Int64:
		return bi.i.Cmp(big.NewInt(n.v.I64())), nil

	case Float64:
		var f big.Float
		return f.SetInt(bi.i).Cmp(big.NewFloat(n.v.F64())), nil

	case BigInt:
		return bi.i.Cmp(n.i), nil

	case BigFloat:
		var f big.Float
		return f.SetInt(bi.i).Cmp(n.f), nil

	case Frac:
		var r big.Rat
		return r.SetFrac(bi.i, &unit).Cmp(n.r), nil

	default:
		return 0, ErrIncomparableTypes
	}
}

// Float64 represents a 64-bit floating-point number
type Float64 struct {
	v api.Value
}

// NewFloat64 .
func NewFloat64(a capnp.Arena, f float64) (f64 Float64, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if f64.v, err = api.NewRootValue(seg); err == nil {
		f64.v.SetF64(f)
	}

	return
}

// Value for Float64 type
func (f64 Float64) Value() api.Value {
	return f64.v
}

// SExpr returns a valid s-expression for Float64
func (f64 Float64) SExpr() (string, error) {
	return strconv.FormatFloat(f64.v.F64(), 'g', -1, 64), nil
}

// Comp returns 0 if the v == other, -1 if v < other, and 1 if v > other.
func (f64 Float64) Comp(other parens.Any) (int, error) {
	switch n := other.(type) {
	case Int64:
		var f big.Float
		return big.NewFloat(f64.v.F64()).Cmp(f.SetInt64(n.v.I64())), nil

	case Float64:
		return compF64(f64.v.F64(), n.v.F64()), nil

	case BigInt:
		var f big.Float
		return big.NewFloat(f64.v.F64()).Cmp(f.SetInt(n.i)), nil

	case BigFloat:
		var bi big.Float
		bi.SetFloat64(f64.v.F64())
		return bi.Cmp(n.f), nil

	case Frac:
		var r big.Rat
		r.SetFloat64(f64.v.F64())
		return r.Cmp(n.r), nil

	default:
		return 0, ErrIncomparableTypes

	}
}

// BigFloat represents an arbitrary-precision floating-point number.
type BigFloat struct {
	f *big.Float
	v api.Value
}

// NewBigFloat .
func NewBigFloat(a capnp.Arena, f *big.Float) (bf BigFloat, err error) {
	bf.f = f

	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if bf.v, err = api.NewRootValue(seg); err == nil {
		err = bf.v.SetBigFloat(f.Text('g', -1))
	}

	return
}

// Value for BigFloat type
func (bf BigFloat) Value() api.Value {
	return bf.v
}

func bigFloatFromValue(v api.Value) (bf BigFloat, err error) {
	bf.f = &big.Float{}
	bf.v = v

	var s string
	if s, err = v.BigFloat(); err == nil {
		if _, ok := bf.f.SetString(s); !ok {
			err = fmt.Errorf("invalid bigfloat format '%s'", s)
		}
	}

	return
}

// SExpr returns a valid s-expression for Float64
func (bf BigFloat) SExpr() (string, error) {
	return bf.f.Text('g', -1), nil
}

// Comp returns 0 if the v == other, -1 if v < other, and 1 if v > other.
func (bf BigFloat) Comp(other parens.Any) (int, error) {
	switch n := other.(type) {
	case Int64:
		var f big.Float
		return bf.f.Cmp(f.SetInt64(n.v.I64())), nil
	case Float64:
		return bf.f.Cmp(big.NewFloat(n.v.F64())), nil
	case BigInt:
		var f big.Float
		return bf.f.Cmp(f.SetInt(n.i)), nil

	case BigFloat:
		return bf.f.Cmp(n.f), nil

	case Frac:
		var r big.Rat
		bf.f.Rat(&r)
		return r.Cmp(n.r), nil

	default:
		return 0, ErrIncomparableTypes

	}
}

// Frac represents a rational number a/b of arbitrary precision.
type Frac struct {
	r *big.Rat
	v api.Value
}

// NewFrac .
func NewFrac(a capnp.Arena, r *big.Rat) (f Frac, err error) {
	f.r = r

	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if f.v, err = api.NewRootValue(seg); err != nil {
		return
	}

	var frac api.Frac
	if frac, err = f.v.NewFrac(); err != nil {
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

func fracFromValue(v api.Value) (f Frac, err error) {
	f.r = &big.Rat{}
	f.v = v

	var fv api.Frac
	if fv, err = v.Frac(); err != nil {
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

// Value for Frac type
func (f Frac) Value() api.Value {
	return f.v
}

// SExpr returns a valid s-expression for frac.
func (f Frac) SExpr() (string, error) {
	return f.r.String(), nil
}

// Comp returns true if the other value is numerical and has the same value.
func (f Frac) Comp(other parens.Any) (int, error) {
	switch n := other.(type) {
	case Int64:
		var r big.Rat
		return f.r.Cmp(r.SetFrac(big.NewInt(n.v.I64()), &unit)), nil

	case Float64:
		var r big.Rat
		return f.r.Cmp(r.SetFloat64(n.v.F64())), nil

	case BigInt:
		var r big.Rat
		return f.r.Cmp(r.SetFrac(n.i, &unit)), nil

	case BigFloat:
		var r big.Rat
		n.f.Rat(&r)
		return f.r.Cmp(&r), nil

	case Frac:
		return f.r.Cmp(n.r), nil

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
