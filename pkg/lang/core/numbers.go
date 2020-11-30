package core

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/mem"
	capnp "zombiezen.com/go/capnproto2"
)

var unit big.Int

func init() { unit.SetInt64(1) }

// Numerical values are comparable.
type Numerical interface {
	ww.Any
	Zero() bool
	Comp(ww.Any) (int, error)
}

// Int64 represents a 64-bit signed integer.
type Int64 interface {
	Numerical
	Int64() int64
}

type i64 struct{ mem.Value }

// NewInt64 .
func NewInt64(a capnp.Arena, i int64) (Int64, error) {
	_, seg, err := capnp.NewMessage(a)
	if err != nil {
		return nil, err
	}

	var res i64
	if res.Raw, err = api.NewRootValue(seg); err == nil {
		res.Raw.SetI64(i)
	}

	return res, nil
}

// Int64 satsifies Int64
func (i i64) Int64() int64 { return i.Raw.I64() }

func (i i64) String() string { return fmt.Sprintf("%d", i.Int64()) }

// Comp returns 0 if the v == other, -1 if v < other, and 1 if v > other.
func (i i64) Comp(other ww.Any) (int, error) {
	switch o := other.MemVal(); o.Type() {
	case api.Value_Which_i64:
		return compI64(i.Int64(), o.Raw.I64()), nil

	case api.Value_Which_f64:
		var f big.Float
		return f.SetInt64(i.Int64()).Cmp(big.NewFloat(o.Raw.F64())), nil

	case api.Value_Which_bigInt:
		return big.NewInt(i.Int64()).Cmp(other.(BigInt).BigInt()), nil

	case api.Value_Which_bigFloat:
		var f big.Float
		return f.SetInt64(i.Int64()).Cmp(other.(BigFloat).f), nil

	case api.Value_Which_frac:
		var r big.Rat
		return r.SetInt64(i.Int64()).Cmp(other.(Fraction).Rat()), nil

	default:
		return 0, ErrIncomparableTypes

	}
}

// Zero returns true if the value is 0.
func (i i64) Zero() bool { return i.Int64() == 0 }

// BigInt represents an arbitrary-length signed integer
type BigInt interface {
	Numerical
	BigInt() *big.Int
}

type bigInt struct {
	i *big.Int
	mem.Value
}

// NewBigInt .
func NewBigInt(a capnp.Arena, i *big.Int) (BigInt, error) {
	_, seg, err := capnp.NewMessage(a)
	if err != nil {
		return nil, err
	}

	var v mem.Value
	if v.Raw, err = api.NewRootValue(seg); err == nil {
		err = v.Raw.SetBigInt(i.Bytes())
	}

	return bigInt{i: i, Value: v}, nil
}

func asBigInt(v mem.Value) (BigInt, error) {
	var i big.Int
	if buf, err := v.Raw.BigInt(); err == nil {
		i.SetBytes(buf)
	}

	return bigInt{i: &i, Value: v}, nil
}

// BigInt satisfies BigInt
func (bi bigInt) BigInt() *big.Int { return bi.i }

func (bi bigInt) String() string { return bi.i.String() }

// Comp returns 0 if the v == other, -1 if v < other, and 1 if v > other.
func (bi bigInt) Comp(other ww.Any) (int, error) {
	switch o := other.MemVal(); o.Type() {
	case api.Value_Which_i64:
		return bi.i.Cmp(big.NewInt(o.Raw.I64())), nil

	case api.Value_Which_f64:
		var f big.Float
		return f.SetInt(bi.i).Cmp(big.NewFloat(o.Raw.F64())), nil

	case api.Value_Which_bigInt:
		return bi.i.Cmp(other.(BigInt).BigInt()), nil

	case api.Value_Which_bigFloat:
		var f big.Float
		return f.SetInt(bi.i).Cmp(other.(BigFloat).f), nil

	case api.Value_Which_frac:
		var r big.Rat
		return r.SetFrac(bi.i, &unit).Cmp(other.(Fraction).Rat()), nil

	default:
		return 0, ErrIncomparableTypes
	}
}

// Zero returns true if the value is 0.
func (bi bigInt) Zero() bool { return bi.i.Sign() == 0 }

// Float64 represents a 64-bit floating-point number
type Float64 interface {
	Numerical
	Float64() float64
}

type f64 struct{ mem.Value }

// NewFloat64 .
func NewFloat64(a capnp.Arena, f float64) (Float64, error) {
	_, seg, err := capnp.NewMessage(a)
	if err != nil {
		return nil, err
	}

	var v mem.Value
	if v.Raw, err = api.NewRootValue(seg); err == nil {
		v.Raw.SetF64(f)
	}

	return f64{v}, nil
}

// Float64 satisfies Float64
func (f f64) Float64() float64 { return f.Raw.F64() }

func (f f64) String() string { return strconv.FormatFloat(f.Float64(), 'g', -1, 64) }

// Comp returns 0 if the v == other, -1 if v < other, and 1 if v > other.
func (f f64) Comp(other ww.Any) (int, error) {
	switch o := other.MemVal(); o.Type() {
	case api.Value_Which_i64:
		var bf big.Float
		return big.NewFloat(f.Float64()).Cmp(bf.SetInt64(o.Raw.I64())), nil

	case api.Value_Which_f64:
		return compF64(f.Float64(), o.Raw.F64()), nil

	case api.Value_Which_bigInt:
		var bf big.Float
		return big.NewFloat(f.Float64()).Cmp(bf.SetInt(other.(BigInt).BigInt())), nil

	case api.Value_Which_bigFloat:
		var bi big.Float
		bi.SetFloat64(f.Float64())
		return bi.Cmp(other.(BigFloat).f), nil

	case api.Value_Which_frac:
		var r big.Rat
		r.SetFloat64(f.Float64())
		return r.Cmp(other.(Fraction).Rat()), nil

	default:
		return 0, ErrIncomparableTypes

	}
}

// Zero returns true if the value is 0.
func (f f64) Zero() bool { return f.Float64() == 0 }

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

// BigFloat satisfies BigFloat
func (bf BigFloat) BigFloat() *big.Float { return bf.f }

func (bf BigFloat) String() string { return bf.f.Text('g', -1) }

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
		return bf.f.Cmp(f.SetInt(other.(BigInt).BigInt())), nil

	case api.Value_Which_bigFloat:
		return bf.f.Cmp(other.(BigFloat).f), nil

	case api.Value_Which_frac:
		var r big.Rat
		bf.f.Rat(&r)
		return r.Cmp(other.(Fraction).Rat()), nil

	default:
		return 0, ErrIncomparableTypes

	}
}

// Zero returns true if the value is 0.
func (bf BigFloat) Zero() bool { return bf.f.Sign() == 0 }

// Fraction represents a rational number a/b of arbitrary precision.
type Fraction interface {
	Numerical
	Rat() *big.Rat
}

type frac struct {
	r *big.Rat
	mem.Value
}

// NewFraction with built-in implementation.
func NewFraction(a capnp.Arena, r *big.Rat) (Fraction, error) {
	_, seg, err := capnp.NewMessage(a)
	if err != nil {
		return nil, err
	}

	var v mem.Value
	if v.Raw, err = api.NewRootValue(seg); err != nil {
		return nil, err
	}

	var f api.Frac
	if f, err = v.Raw.NewFrac(); err != nil {
		return nil, err
	}

	if err = f.SetNumer(r.Num().Bytes()); err != nil {
		return nil, err
	}

	if err = f.SetDenom(r.Denom().Bytes()); err != nil {
		return nil, err
	}

	return frac{r: r, Value: v}, nil
}

func asFrac(v mem.Value) (Fraction, error) {
	fv, err := v.Raw.Frac()
	if err != nil {
		return nil, err
	}

	var nbuf, dbuf []byte
	if nbuf, err = fv.Numer(); err != nil {
		return nil, err
	}

	if dbuf, err = fv.Denom(); err != nil {
		return nil, err
	}

	var numer, denom big.Int
	numer.SetBytes(nbuf)
	denom.SetBytes(dbuf)

	var r big.Rat
	return frac{
		r:     r.SetFrac(&numer, &denom),
		Value: v,
	}, nil
}

// Rat satisfies Fraction
func (f frac) Rat() *big.Rat { return f.r }

func (f frac) String() string { return f.r.String() }

// Comp returns true if the other value is numerical and has the same value.
func (f frac) Comp(other ww.Any) (int, error) {
	switch o := other.MemVal(); o.Type() {
	case api.Value_Which_i64:
		var r big.Rat
		return f.r.Cmp(r.SetFrac(big.NewInt(o.Raw.I64()), &unit)), nil

	case api.Value_Which_f64:
		var r big.Rat
		return f.r.Cmp(r.SetFloat64(o.Raw.F64())), nil

	case api.Value_Which_bigInt:
		var r big.Rat
		return f.r.Cmp(r.SetFrac(other.(BigInt).BigInt(), &unit)), nil

	case api.Value_Which_bigFloat:
		var r big.Rat
		other.(BigFloat).f.Rat(&r)
		return f.r.Cmp(&r), nil

	case api.Value_Which_frac:
		return f.r.Cmp(other.(Fraction).Rat()), nil

	default:
		return 0, ErrIncomparableTypes

	}
}

// Zero returns true if the value is 0.
func (f frac) Zero() bool { return f.r.Sign() == 0 }

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
