package query

import "github.com/wetware/ww/cluster/routing"

type Query struct {
	routing.Snapshot
}

// Reverse returns a Query that traverses the snapshot in
// reverse lexicographical-order.  Calling Reverse on the
// resulting query undoes the reversal.
func (q Query) Reverse() Query {
	if r, ok := q.Snapshot.(reversed); ok {
		return Query(r)
	}

	return Query{Snapshot: reversed(q)}
}

// Lookup returns the first record in the selection.  The
// returned record is nil if the selection is empty.
func (q Query) Lookup(sel Selector, cs ...Constraint) (routing.Record, error) {
	it, err := q.Iter(sel, cs...)
	if it == nil || err != nil {
		return nil, err
	}

	return it.Next(), nil
}

// Iter traverses the snapshot in lexicographical order,
// unless q is the result of having called Reverse() on
// a non-reversed Query.
func (q Query) Iter(sel Selector, cs ...Constraint) (routing.Iterator, error) {
	for _, constraint := range cs {
		sel = sel.Bind(constraint)
	}

	return sel(q.Snapshot)
}

type reversed struct{ routing.Snapshot }

func (r reversed) Get(ix routing.Index) (routing.Iterator, error) {
	return r.Snapshot.GetReverse(ix)
}

func (r reversed) GetReverse(ix routing.Index) (routing.Iterator, error) {
	return r.Snapshot.Get(ix)
}

func (r reversed) LowerBound(ix routing.Index) (routing.Iterator, error) {
	return r.Snapshot.ReverseLowerBound(ix)
}

func (r reversed) ReverseLowerBound(ix routing.Index) (routing.Iterator, error) {
	return r.Snapshot.LowerBound(ix)
}
