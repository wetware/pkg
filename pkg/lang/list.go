package lang

import (
	"github.com/spy16/parens"
	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/mem"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	// EmptyList is a zero-value List
	EmptyList List

	_ ww.Any     = (*List)(nil)
	_ parens.Seq = (*List)(nil)
)

func init() {
	_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		panic(err)
	}

	if EmptyList.Raw, err = api.NewRootValue(seg); err != nil {
		panic(err)
	}

	if _, err = EmptyList.Raw.NewList(); err != nil {
		panic(err)
	}
}

// List is a persistent, singly-linked list with fast insertions/pops to its head.
type List struct{ mem.Value }

// NewList returns a new list containing given values.
func NewList(a capnp.Arena, vs ...parens.Any) (l List, err error) {
	if len(vs) == 0 {
		return EmptyList, nil
	}

	if l, _, err = newList(a); err != nil {
		return
	}

	for i := len(vs) - 1; i >= 0; i-- {
		l, err = listCons(capnp.SingleSegment(nil), vs[i].(ww.Any).Data(), l)
		if err != nil {
			break
		}
	}

	return
}

// Count returns the number of the list.
func (l List) Count() (int, error) {
	ll, _, err := listIsNull(l.Raw)
	return int(ll.Count()), err
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
		if res, err = res.Cons(item.(ww.Any)); err != nil {
			return nil, err
		}
	}

	return res, nil
}

// Cons returns a new list with the item added at the head of the list.
func (l List) Cons(any parens.Any) (List, error) {
	return listCons(capnp.SingleSegment(nil), any.(ww.Any).Data(), l)
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
func (l List) Next() (parens.Seq, error) {
	_, next, err := listNext(l.Raw)
	return next, err
}

func (l List) count() (ll api.LinkedList, cnt int, err error) {
	if ll, err = l.Raw.List(); err == nil {
		cnt = int(ll.Count())
	}

	return
}

func (l List) head() (ll api.LinkedList, v parens.Any, err error) {
	if ll, err = l.Raw.List(); err != nil {
		return
	}

	var val mem.Value
	if val.Raw, err = ll.Head(); err == nil {
		v, err = AsAny(val)
	}

	return
}

func (l List) isNull() (null bool, err error) {
	_, null, err = listIsNull(l.Raw)
	return
}

func listTail(v mem.Value) (ll api.LinkedList, tail List, err error) {
	if ll, err = v.Raw.List(); err != nil {
		return
	}

	tail.Raw, err = ll.Tail()
	return
}

// func listToSlice(l List, sizeHint int) ([]parens.Any, error) {
// 	slice := make([]parens.Any, 0, sizeHint)
// 	err := parens.ForEach(l, func(item parens.Any) (bool, error) {
// 		slice = append(slice, item)
// 		return false, nil
// 	})
// 	return slice, err
// }

func listCons(a capnp.Arena, v mem.Value, tail List) (l List, err error) {
	var ll api.LinkedList
	if l, ll, err = newList(a); err != nil {
		return
	}

	if err = ll.SetHead(v.Raw); err != nil {
		return
	}

	if err = ll.SetTail(tail.Raw); err != nil {
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

func listIsNull(v api.Value) (l api.LinkedList, null bool, err error) {
	l, err = v.List()
	null = err == nil && l.Count() == 0
	return
}

func listNext(v api.Value) (api.LinkedList, parens.Seq, error) {
	ll, null, err := listIsNull(v)
	if err != nil || null {
		return ll, nil, err
	}

	var l List
	if l.Raw, err = ll.Tail(); err != nil {
		return ll, nil, err
	}

	return ll, l, nil
}

func newList(a capnp.Arena) (l List, ll api.LinkedList, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if l.Raw, err = api.NewRootValue(seg); err != nil {
		return
	}

	if ll, err = l.Raw.NewList(); err != nil {
		return
	}

	return
}
