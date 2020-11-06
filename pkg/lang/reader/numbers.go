package reader

import (
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"strconv"
	"strings"

	score "github.com/spy16/slurp/core"
	"github.com/spy16/slurp/reader"

	capnp "zombiezen.com/go/capnproto2"

	"github.com/wetware/ww/pkg/lang/builtin"
	"github.com/wetware/ww/pkg/lang/core"
)

// import (
// 	"fmt"
// 	"io"
// 	"math"
// 	"math/big"
// 	"strconv"
// 	"strings"

// 	"github.com/pkg/errors"
//
// 	"github.com/spy16/slurp/reader"
// 	"github.com/wetware/ww/pkg/lang"
// 	"github.com/wetware/ww/pkg/lang/builtin"
// 	capnp "zombiezen.com/go/capnproto2"
// )

func readNumber(rd *reader.Reader, init rune) (v score.Any, err error) {
	beginPos := rd.Position()

	numStr, err := readNumToken(rd, init)
	if err != nil {
		return nil, err
	}

	decimalPoint := strings.ContainsRune(numStr, '.')
	isRadix := strings.ContainsRune(numStr, 'r')
	isScientific := strings.ContainsRune(numStr, 'e')
	isFrac := strings.ContainsRune(numStr, '/')

	switch {
	case isRadix && (decimalPoint || isScientific || isFrac):
		err = reader.ErrNumberFormat

	case isScientific:
		v, err = parseScientific(numStr)

	case decimalPoint:
		v, err = parseFloat(numStr)

	case isRadix:
		v, err = parseRadix(numStr)

	case isFrac:
		v, err = parseFrac(numStr)

	default:
		v, err = parseInt(numStr)

	}

	if err != nil {
		err = annotateErr(rd, err, beginPos, numStr)
	}

	return
}

func parseInt(numStr string) (score.Any, error) {
	v, err := strconv.ParseInt(numStr, 0, 64)
	switch {
	case err == nil:
		// TODO(performance):  pre-allocate arena
		return builtin.NewInt64(capnp.SingleSegment(nil), v)

	case errors.Is(err, strconv.ErrRange):
		var b big.Int
		if _, ok := b.SetString(numStr, 0); !ok {
			return nil, fmt.Errorf("%w (bigint): '%s'", reader.ErrNumberFormat, numStr)
		}

		// TODO(performance):  pre-allocate arena
		return builtin.NewBigInt(capnp.SingleSegment(nil), &b)
	default:
		return nil, fmt.Errorf("%w (int64): '%s'", reader.ErrNumberFormat, numStr)

	}
}

func parseFloat(numStr string) (score.Any, error) {
	v, err := strconv.ParseFloat(numStr, 64)
	switch {
	case err == nil:
		// TODO(performance):  pre-allocate arena
		return builtin.NewFloat64(capnp.SingleSegment(nil), v)

	case errors.Is(err, strconv.ErrRange):
		var f big.Float
		if _, ok := f.SetString(numStr); !ok {
			return nil, fmt.Errorf("%w (bigfloat): '%s'", reader.ErrNumberFormat, numStr)
		}

		// TODO(performance):  pre-allocate arena
		return builtin.NewBigFloat(capnp.SingleSegment(nil), &f)

	default:
		return nil, reader.ErrNumberFormat

	}
}

func parseRadix(numStr string) (core.Int64, error) {
	parts := strings.Split(numStr, "r")
	if len(parts) != 2 {
		return nil, fmt.Errorf("%w (radix notation): '%s'", reader.ErrNumberFormat, numStr)
	}

	base, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w (radix notation): '%s'", reader.ErrNumberFormat, numStr)
	}

	repr := parts[1]
	if base < 0 {
		base = -1 * base
		repr = "-" + repr
	}

	v, err := strconv.ParseInt(repr, int(base), 64)
	if err != nil {
		return nil, fmt.Errorf("%w (radix notation): '%s'", reader.ErrNumberFormat, numStr)
	}

	return builtin.NewInt64(capnp.SingleSegment(nil), v)
}

func parseScientific(numStr string) (score.Any, error) {
	parts := strings.Split(numStr, "e")
	if len(parts) != 2 {
		return nil, fmt.Errorf("%w (scientific notation): '%s'", reader.ErrNumberFormat, numStr)
	}

	base, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return nil, fmt.Errorf("%w (scientific notation): '%s'", reader.ErrNumberFormat, numStr)
	}

	pow, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w (scientific notation): '%s'", reader.ErrNumberFormat, numStr)
	}

	f := base * math.Pow(10, float64(pow))

	if math.IsInf(f, 0) {
		var bf big.Float
		if _, ok := bf.SetString(numStr); !ok {
			return nil, fmt.Errorf("%w (bigfloat): '%s'", reader.ErrNumberFormat, numStr)
		}

		return builtin.NewBigFloat(capnp.SingleSegment(nil), &bf)
	}

	return builtin.NewFloat64(capnp.SingleSegment(nil), f)
}

func parseFrac(numStr string) (score.Any, error) { // TODO:  return lang.Frac
	parts := strings.Split(numStr, "/")
	if len(parts) != 2 || parts[1] == "" {
		return nil, fmt.Errorf("%w (fractional notation): '%s'", reader.ErrNumberFormat, numStr)
	}

	var numer, denom big.Int
	if _, ok := numer.SetString(parts[0], 0); !ok {
		return nil, fmt.Errorf("%w (numerator): '%s'", reader.ErrNumberFormat, numStr)
	}
	if _, ok := denom.SetString(parts[1], 0); !ok {
		return nil, fmt.Errorf("%w (denominator): '%s'", reader.ErrNumberFormat, numStr)
	}

	var r big.Rat
	return builtin.NewFrac(capnp.SingleSegment(nil), r.SetFrac(&numer, &denom))
}

// Token reads one token from the reader and returns. If init is not -1, it is included
// as first character in the token.
func readNumToken(rd *reader.Reader, init rune) (string, error) {
	var b strings.Builder
	if init != -1 {
		b.WriteRune(init)
	}

	for {
		r, err := rd.NextRune()
		if err != nil {
			if err == io.EOF {
				break
			}
			return b.String(), err
		}

		if r != '/' && rd.IsTerminal(r) {
			rd.Unread(r)
			break
		}

		b.WriteRune(r)
	}

	return b.String(), nil
}
