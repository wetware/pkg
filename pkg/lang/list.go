package lang

import (
	"github.com/spy16/parens"
	"github.com/wetware/ww/internal/api"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	// EmptyList is a zero-value List
	EmptyList List

	_ parens.Any       = (*List)(nil)
	_ parens.Seq       = (*List)(nil)
	_ apiValueProvider = (*List)(nil)
)

func init() {
	_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		panic(err)
	}

	if EmptyList.v, err = api.NewRootValue(seg); err != nil {
		panic(err)
	}

	if _, err = EmptyList.v.NewList(); err != nil {
		panic(err)
	}
}

// List is a persistent, singly-linked list with fast insertions/pops to its head.
type List struct {
	v api.Value
}

// NewList returns a new list containing given values.
func NewList(a capnp.Arena, vs ...parens.Any) (l List, err error) {
	if len(vs) == 0 {
		return EmptyList, nil
	}

	if l, _, err = newList(a); err != nil {
		return
	}

	for i := len(vs) - 1; i >= 0; i-- {
		if l, err = listCons(capnp.SingleSegment(nil), vs[i], l); err != nil {
			break
		}
	}

	return
}

// Count returns the number of the list.
func (l List) Count() (cnt int, err error) {
	var null bool
	if null, err = l.isNull(); err == nil && !null {
		_, cnt, err = l.count()
	}

	return
}

// SExpr returns a valid s-expression for List
func (l List) SExpr() (string, error) {
	return parens.SeqString(l, "(", ")", " ")
}

// Conj returns a new list with all the items added at the head of the list.
func (l List) Conj(items ...parens.Any) (parens.Seq, error) {
	null, err := l.isNull()
	if err != nil {
		return nil, err
	}

	var res List
	if null {
		res = l
	} else {
		res = EmptyList
	}

	for _, item := range items {
		if res, err = res.Cons(item); err != nil {
			return nil, err
		}
	}

	return res, nil
}

// Cons returns a new list with the item added at the head of the list.
func (l List) Cons(v parens.Any) (List, error) {
	return listCons(capnp.SingleSegment(nil), v, l)
}

// First returns the head or first item of the list.
func (l List) First() (v parens.Any, err error) {
	var null bool
	if null, err = l.isNull(); err == nil && !null {
		_, v, err = l.head()
	}

	return
}

// Next returns the tail of the list.
func (l List) Next() (tail parens.Seq, err error) {
	tail, err = l.Tail()
	return
}

// Tail returns the tail of the list.  It is equivalent to Next() except that it returns
// a List.
func (l List) Tail() (tail List, err error) {
	var null bool
	if null, err = l.isNull(); err == nil && !null {
		_, tail, err = l.tail()
	}

	return
}

// Value returns the api.Value for List
func (l List) Value() api.Value {
	return l.v
}

func (l List) count() (ll api.LinkedList, cnt int, err error) {
	if ll, err = l.v.List(); err == nil {
		cnt = int(ll.Count())
	}

	return
}

func (l List) head() (ll api.LinkedList, v parens.Any, err error) {
	if ll, err = l.v.List(); err != nil {
		return
	}

	var raw api.Value
	if raw, err = ll.Head(); err != nil {
		return
	}

	v, err = valueOf(raw)
	return
}

func (l List) tail() (ll api.LinkedList, tail List, err error) {
	if ll, err = l.v.List(); err != nil {
		return
	}

	var val api.Value
	if val, err = ll.Tail(); err == nil {
		tail = List{v: val} // TODO(xxx) position?
	}

	return
}

func (l List) isNull() (bool, error) {
	lv, err := l.v.List()
	if err != nil {
		return false, err
	}

	return lv.Count() == 0, nil
}

// func listToSlice(l List, sizeHint int) ([]parens.Any, error) {
// 	slice := make([]parens.Any, 0, sizeHint)
// 	err := parens.ForEach(l, func(item parens.Any) (bool, error) {
// 		slice = append(slice, item)
// 		return false, nil
// 	})
// 	return slice, err
// }

func listCons(a capnp.Arena, v parens.Any, tail List) (l List, err error) {
	var ll api.LinkedList
	if l, ll, err = newList(a); err != nil {
		return
	}

	if err = ll.SetHead(v.(apiValueProvider).Value()); err != nil {
		return
	}

	if err = ll.SetTail(tail.v); err != nil {
		return
	}

	var null bool
	if null, err = tail.isNull(); err != nil {
		return
	}

	var cnt int = 1
	if !null {
		if cnt, err = tail.Count(); err != nil {
			return
		}

		cnt++
	}

	ll.SetCount(uint32(cnt))
	return
}

func newList(a capnp.Arena) (l List, ll api.LinkedList, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if l.v, err = api.NewRootValue(seg); err != nil {
		return
	}

	if ll, err = l.v.NewList(); err != nil {
		return
	}

	return
}
