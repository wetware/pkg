// Package lang contains the wetware language iplementation
package lang

import (
	"github.com/pkg/errors"
	"github.com/spy16/parens"
)

var (
	globals = map[string]parens.Any{
		"nil":   Nil{},
		"true":  True,
		"false": False,
	}
)

// New returns a new root environment.
func New() *parens.Env {
	return parens.New(
		parens.WithGlobals(globals, nil),
		// parens.WithAnalyzer(...),
	)
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
