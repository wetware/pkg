// Package lang contains the wetware language iplementation
package lang

import (
	"github.com/pkg/errors"
	"github.com/spy16/parens"
	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	capnp "zombiezen.com/go/capnproto2"
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
	if a == nil {
		panic("nil Anchor")
	}

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
func Pop(col ww.Any) (parens.Any, error) {
	switch col.Value().Which() {
	case api.Value_Which_list:
		_, tail, err := listTail(col.Value())
		return tail, err

	case api.Value_Which_vector:
		_, vec, err := vectorPop(col.Value())
		return vec, err

	default:
		return nil, parens.Error{
			Cause:   errors.New("unordered collection or atom"),
			Message: col.Value().Which().String(),
		}

	}
}

// Conj returns a new collection with the xs
// 'added'. (conj nil item) returns (item).  The 'addition' may
// happen at different 'places' depending on the concrete type.
func Conj(col ww.Any, vs parens.Seq) (parens.Any, error) {
	switch col.Value().Which() {
	case api.Value_Which_list:
		l := List{col.Value()}
		err := parens.ForEach(vs, func(v parens.Any) (_ bool, err error) {
			l, err = listCons(capnp.SingleSegment(nil), v.(ww.Any).Value(), l)
			return
		})
		return l, err

	case api.Value_Which_vector:
		vec := Vector{col.Value()}
		err := parens.ForEach(vs, func(v parens.Any) (_ bool, err error) {
			vec, err = vec.Conj(v)
			return
		})
		return vec, err

	default:
		return nil, parens.Error{
			Cause:   errors.New("unordered collection or atom"),
			Message: col.Value().Which().String(),
		}

	}
}
