package core

import (
	"fmt"
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
	mv, err := mem.NewValue(capnp.SingleSegment(nil))
	if err != nil {
		panic(err)
	}

	if _, err = mv.MemVal().NewList(); err != nil {
		panic(err)
	}

	EmptyList.Value = mv
}

// List is a persistent, singly-linked list with fast insertions/pops to its head.
type List interface {
	ww.Any
	Seq
	Count() (int, error)
	Cons(any ww.Any) (List, error)
}

type list struct{ mem.Value }

// NewList returns a new list containing given values.
func NewList(a capnp.Arena, vs ...ww.Any) (l List, err error) {
	if len(vs) == 0 {
		return EmptyList, nil
	}

	if l, _, err = newList(a); err != nil {
		return nil, err
	}

	for i := len(vs) - 1; i >= 0; i-- {
		l, err = Cons(capnp.SingleSegment(nil), vs[i], l)
		if err != nil {
			break
		}
	}

	return l, err
}

// Count returns the number of the list.
func (l list) Count() (int, error) {
	ll, err := l.MemVal().List()
	return int(ll.Count()), err
}

// Render the list into human-readable form
func (l list) Render() (string, error) {
	return l.render(func(any ww.Any) (string, error) {
		return Render(any.(ww.Any))
	})
}

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
func (l list) Conj(items ...ww.Any) (Container, error) {
	ll, err := l.MemVal().List()
	if err != nil {
		return nil, err
	}

	var res List
	if l.isNull(ll) {
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
	return Cons(capnp.SingleSegment(nil), any, l)
}

// First returns the head or first item of the list.
func (l list) First() (ww.Any, error) {
	ll, err := l.MemVal().List()
	if err != nil || l.isNull(ll) {
		return nil, err
	}

	return l.head(ll)
}

// Next returns the tail of the list.
func (l list) Next() (Seq, error) {
	ll, err := l.MemVal().List()
	if err != nil {
		return nil, err
	}

	next, err := l.next(ll)
	if err == ErrIllegalState { // (next ' ()) => nil
		return nil, nil
	}

	return next, err
}

func (l list) isNull(ll api.LinkedList) bool { return ll.Count() == 0 }

func (l list) head(ll api.LinkedList) (v ww.Any, err error) {
	var val api.Any
	if val, err = ll.Head(); err == nil {
		v, err = AsAny(val)
	}

	return
}

func (l list) next(ll api.LinkedList) (Seq, error) {
	if l.isNull(ll) {
		return nil, ErrIllegalState
	}

	val, err := ll.Tail()
	if err != nil {
		return nil, err
	}

	any, err := AsAny(val)
	if err != nil {
		return nil, err
	}

	if seq, ok := any.(Seq); ok {
		return seq, nil
	}

	return nil, fmt.Errorf("%w: non-sequence type '%T' in tail",
		ErrIllegalState, any)
}

func newList(a capnp.Arena) (l list, ll api.LinkedList, err error) {
	if l.Value, err = mem.NewValue(a); err == nil {
		ll, err = l.MemVal().NewList()
	}

	return
}
