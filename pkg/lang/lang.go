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

// Pop returns a collection without one item.  For a list, Pop
// returns a new list/queue without the first item, for a vector,
// returns a new vector without the last item. If the collection
// is empty, returns an error.
func Pop(v parens.Any) (parens.Any, error) {
	switch c := v.(type) {
	case List:
		return c.Tail()

	case Vector:
		return c.Pop()

	default:
		return nil, parens.Error{
			Cause:   errors.New("unordered collection or atom"),
			Message: reflect.TypeOf(v).String(),
		}

	}
}

// Conj returns a new collection with the xs
// 'added'. (conj nil item) returns (item).  The 'addition' may
// happen at different 'places' depending on the concrete type.
func Conj(v parens.Any, vs parens.Seq) (parens.Any, error) {
	switch c := v.(type) {
	case List:
		err := parens.ForEach(vs, func(v parens.Any) (_ bool, err error) {
			c, err = c.Cons(v)
			return
		})
		return c, err

	case Vector:
		err := parens.ForEach(vs, func(v parens.Any) (_ bool, err error) {
			c, err = c.Conj(v)
			return
		})
		return c, err

	default:
		return nil, parens.Error{
			Cause:   errors.New("unordered collection or atom"),
			Message: reflect.TypeOf(v).String(),
		}

	}
}
