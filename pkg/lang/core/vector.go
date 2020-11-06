package core

import (
	ww "github.com/wetware/ww/pkg"
)

// Vector is a persistent, ordered collection of values with fast random lookups and
// insertions.
type Vector interface {
	ww.Any
	Count() (int, error)
	Conj(ww.Any) (Vector, error)
	EntryAt(i int) (ww.Any, error)
	Assoc(i int, val ww.Any) (Vector, error)
	Pop() (Vector, error)
}
