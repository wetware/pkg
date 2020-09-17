// Package lang contains the wetware language iplementation
package lang

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/spy16/parens"
	ww "github.com/wetware/ww/pkg"
)

var (
	globals = map[string]parens.Any{
		"nil":   Nil{},
		"true":  True,
		"false": False,
	}
)

// New returns a new root environment.
func New(a ww.Anchor) *parens.Env {
	return parens.New(
		parens.WithAnalyzer(analyzer(a)),
		parens.WithGlobals(globals, nil))
}

// ErrIncomparableTypes is returned if two types cannot be meaningfully
// compared to each other.
var ErrIncomparableTypes = errors.New("incomparable types")

// Comparable type.
type Comparable interface {
	// Comp compares the magnitude of the comparable c with that of other.
	// It returns 0 if the magnitudes are equal, -1 if c < other, and 1 if c > other.
	Comp(other parens.Any) (int, error)
}

// Pop removes an element from the collection and returns it, along with a new
// collection of identical type, corresponding to v without the returned value.
// Calling Pop on an atom is an error.
func Pop(v parens.Any) (res parens.Any, col parens.Any, err error) {
	switch c := v.(type) {
	case parens.Seq:
		res, col, err = popSeq(c)

	case Vector:
		var i int
		if i, err = c.Count(); err != nil {
			return
		}

		if res, err = c.EntryAt(i - 1); err == nil {
			col, err = c.Pop()
		}

	default:
		err = parens.Error{
			Cause:   errors.New("not a collection"),
			Message: reflect.TypeOf(v).String(),
		}

	}

	return
}

func popSeq(seq parens.Seq) (res parens.Any, _ parens.Seq, err error) {
	if res, err = seq.First(); err == nil {
		seq, err = seq.Next()
	}

	return res, seq, err
}
