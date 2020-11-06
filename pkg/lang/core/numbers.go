package core

import (
	"math/big"

	ww "github.com/wetware/ww/pkg"
)

// Numerical value.
type Numerical interface {
	ww.Any
	Comparable
}

// Int64 is a fixed-size, 64-bit integer.
type Int64 interface {
	Numerical
	Int64() int64
}

// Float64 is a fixed-size, 64-bit floating-point number.
type Float64 interface {
	Numerical
	Float64() float64
}

// Fraction is an arbitrary-precision rational number.
type Fraction interface {
	Numerical
	Rat() *big.Rat
}

// BigInt is an arbitrary-precision integer.
type BigInt interface {
	Numerical
	BigInt() *big.Int
}

// BigFloat is an arbitrary-precision floating-point number.
type BigFloat interface {
	Numerical
	BigFloat() *big.Float
}
