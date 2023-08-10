package query

import (
	"fmt"

	"github.com/wetware/pkg/cluster/routing"
)

// Selector specifies an iteration strategy over a set of
// routing records.
type Selector func(routing.Snapshot) (routing.Iterator, error)

// Bind a constraint to the selector, causing it to select a
// subset of the current selection.
func (selection Selector) Bind(f Constraint) Selector {
	return func(q routing.Snapshot) (it routing.Iterator, err error) {
		if it, err = selection(q); err == nil {
			it, err = f(it)(q)
		}

		return
	}
}

// All selects all records in the routing table.
func All() Selector {
	return Select(all{})
}

// Select all records matching the index.
func Select(index routing.Index) Selector {
	return func(q routing.Snapshot) (routing.Iterator, error) {
		return q.Get(index)
	}
}

// Select all records, beginning with the indexed record, and
// iterating in lexicographic order.
func From(index routing.Index) Selector {
	return func(q routing.Snapshot) (routing.Iterator, error) {
		return q.LowerBound(index)
	}
}

// Range selects all records in the interval [min, max].
func Range(min, max routing.Index) Selector {
	return From(min).Bind(To(max))
}

// Failure returns a Selector that fails with the supplied error.
func Failure(err error) Selector {
	return func(q routing.Snapshot) (routing.Iterator, error) {
		return nil, err
	}
}

// Failuref formats a string using fmt.Errorf and passes the
// result to Failure.
func Failuref(format string, args ...any) Selector {
	return Failure(fmt.Errorf(format, args...))
}

type filterIter struct {
	Matcher
	routing.Iterator
}

func (it *filterIter) Next() (r routing.Record) {
	for r = it.Iterator.Next(); r != nil && !it.Match(r); r = it.Iterator.Next() {
	}

	return
}

// predicateIter is short-circuits when the Matcher returns false.
// This is more efficient than using filterIter in cases where the
// iterator should stop early.
type predicateIter struct {
	Matcher
	routing.Iterator
	stop bool
}

func (it *predicateIter) Next() (r routing.Record) {
	if !it.stop {
		r = it.Iterator.Next()
		if it.stop = !it.Match(r); it.stop {
			r = nil
		}
	}

	return
}

type all struct{}

func (all) String() string             { return "id" }
func (all) Prefix() bool               { return true }
func (all) PeerBytes() ([]byte, error) { return nil, nil }
