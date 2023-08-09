package routing

import (
	"github.com/hashicorp/go-memdb"
	"github.com/wetware/ww/util/stm"
)

type query struct {
	records stm.TableRef
	tx      stm.Txn
}

func (q query) Get(ix Index) (Iterator, error) {
	return q.iterate(q.tx.Get, ix)
}

func (q query) GetReverse(ix Index) (Iterator, error) {
	return q.iterate(q.tx.GetReverse, ix)
}

func (q query) LowerBound(ix Index) (Iterator, error) {
	return q.iterate(q.tx.LowerBound, ix)
}

func (q query) ReverseLowerBound(ix Index) (Iterator, error) {
	return q.iterate(q.tx.ReverseLowerBound, ix)
}

func (q query) iterate(f iterFunc, ix Index) (Iterator, error) {
	it, err := f(q.records, index(ix), ix)
	return iterator{it}, err
}

func index(ix Index) string {
	if ix.Prefix() {
		return ix.String() + "_prefix"
	}

	return ix.String()
}

type iterFunc func(stm.TableRef, string, ...any) (memdb.ResultIterator, error)

type iterator struct{ memdb.ResultIterator }

func (it iterator) Next() Record {
	if v := it.ResultIterator.Next(); v != nil {
		return v.(Record)
	}

	return nil
}
