package core

import (
	"strings"

	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/mem"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	// EmptyList is a zero-value List
	EmptyList list

	_ ww.Any = (*list)(nil)
	_ Seq    = (*list)(nil)
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

type list struct{ mem.Value }

// NewList returns a new list containing given values.
func NewList(a capnp.Arena, vs ...ww.Any) (List, error) {
	if len(vs) == 0 {
		return EmptyList, nil
	}

	l, _, err := newList(a)
	if err != nil {
		return nil, err
	}

	for i := len(vs) - 1; i >= 0; i-- {
		l, err = listCons(capnp.SingleSegment(nil), vs[i].(ww.Any).MemVal(), l)
		if err != nil {
			break
		}
	}

	return l, err
}

// Count returns the number of the list.
func (l list) Count() (int, error) {
	ll, _, err := listIsNull(l.Raw)
	return int(ll.Count()), err
}

// Render the list into human-readable form
func (l list) Render() (string, error) {
	return l.render(func(any ww.Any) (string, error) {
		return Render(any.(ww.Any))
	})
}

// // SExpr returns a valid s-expression for List
// func (l list) SExpr() (string, error) {
// 	return l.render(func(any ww.Any) (string, error) {
// 		if r, ok := any.(SExpressable); ok {
// 			return r.SExpr()
// 		}

// 		return "", errors.Errorf("%s is not a symbol provider", reflect.TypeOf(any))
// 	})
// }

func (l list) render(f func(ww.Any) (string, error)) (string, error) {
	cnt, err := l.Count()
	if err != nil {
		return "", err
	}

	if cnt == 0 {
		return "()", nil
	}

	var b strings.Builder
	b.WriteRune('(')

	seq := Seq(l)
	for i := 0; i < cnt; i++ {
		if i > 0 {
			b.WriteRune(' ')
		}

		v, err := seq.First()
		if err != nil {
			return "", err
		}

		s, err := f(v)
		if err != nil {
			return "", err
		}

		b.WriteString(s)

		if seq, err = seq.Next(); err != nil {
			return "", err
		}
	}

	b.WriteRune(')')
	return b.String(), nil
}

// Conj returns a new list with all the items added at the head of the list.
func (l list) Conj(items ...ww.Any) (Seq, error) {
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
func (l list) Cons(any ww.Any) (List, error) {
	return listCons(capnp.SingleSegment(nil), any.(ww.Any).MemVal(), l)
}

// First returns the head or first item of the list.
func (l list) First() (v ww.Any, err error) {
	var null bool
	if null, err = l.isNull(); err == nil && !null {
		_, v, err = l.head()
	}

	return
}

// Next returns the tail of the list.
func (l list) Next() (Seq, error) {
	_, next, err := listNext(l.Raw)
	return next, err
}

func (l list) count() (ll api.LinkedList, cnt int, err error) {
	if ll, err = l.Raw.List(); err == nil {
		cnt = int(ll.Count())
	}

	return
}

func (l list) head() (ll api.LinkedList, v ww.Any, err error) {
	if ll, err = l.Raw.List(); err != nil {
		return
	}

	var val mem.Value
	if val.Raw, err = ll.Head(); err == nil {
		v, err = AsAny(val)
	}

	return
}

func (l list) isNull() (null bool, err error) {
	_, null, err = listIsNull(l.Raw)
	return
}

func listTail(v mem.Value) (ll api.LinkedList, tail list, err error) {
	if ll, err = v.Raw.List(); err != nil {
		return
	}

	tail.Raw, err = ll.Tail()
	return
}

// func listToSlice(l List, sizeHint int) ([]ww.Any, error) {
// 	slice := make([]ww.Any, 0, sizeHint)
// 	err := ForEach(l, func(item ww.Any) (bool, error) {
// 		slice = append(slice, item)
// 		return false, nil
// 	})
// 	return slice, err
// }

func listCons(a capnp.Arena, v mem.Value, tail list) (l list, err error) {
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

func listNext(v api.Value) (api.LinkedList, Seq, error) {
	ll, null, err := listIsNull(v)
	if err != nil || null {
		return ll, nil, err
	}

	var l list
	if l.Raw, err = ll.Tail(); err != nil {
		return ll, nil, err
	}

	return ll, l, nil
}

func newList(a capnp.Arena) (l list, ll api.LinkedList, err error) {
	if l.Value, err = mem.NewValue(a); err == nil {
		ll, err = l.Raw.NewList()
	}

	return
}
