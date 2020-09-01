package core

import (
	"fmt"

	"github.com/spy16/sabre/runtime"
	"github.com/wetware/ww/internal/api"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	// EmptyList is a zero-value List
	EmptyList List

	_ runtime.Seq      = (*List)(nil)
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

	EmptyList.v.SetNil()
}

// List is a persistent, singly-linked list with fast insertions/pops to its head.
type List struct {
	runtime.Position
	v api.Value
}

// NewList returns a new list containing given values.
func NewList(a capnp.Arena, vs ...runtime.Value) (l List, err error) {
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
func (l List) Count() int {
	_, cnt, err := l.count()
	if err != nil {
		panic(err)
	}

	return cnt
}

func (l List) String() string {
	return runtime.SeqString(l, "(", ")", " ")
}

// Eval evaluates the first item in the list and invokes the resultant first with
// rest of the list as arguments.
func (l List) Eval(r runtime.Runtime) (runtime.Value, error) {
	if l.isNull() {
		return l, nil
	}

	_, cnt, err := l.count()
	if err != nil {
		return nil, err
	}

	if cnt == 0 {
		return l, nil
	}

	v, err := r.Eval(l.First())
	if err != nil {
		return nil, err
	}

	target, ok := v.(runtime.Invokable)
	if !ok {
		return nil, fmt.Errorf("value of type '%s' is not invokable",
			v.(apiValueProvider).Value().Which())
	}

	return target.Invoke(r, listToSlice(l, cnt-1)...)
}

// Conj returns a new list with all the items added at the head of the list.
func (l List) Conj(items ...runtime.Value) runtime.Seq {
	var res List
	if l.isNull() {
		res = EmptyList
	} else {
		res = l
	}

	var err error
	for _, item := range items {
		if res, err = listCons(capnp.SingleSegment(nil), item, res); err != nil {
			panic(err)
		}
	}

	return res
}

// First returns the head or first item of the list.
func (l List) First() runtime.Value {
	if l.isNull() {
		return nil
	}

	_, v, err := l.head()
	if err != nil {
		panic(err)
	}

	return v
}

// Next returns the tail of the list.
func (l List) Next() runtime.Seq {
	if l.isNull() {
		return nil
	}

	_, tail, err := l.tail()
	if err != nil {
		panic(err)
	}

	return tail
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

func (l List) head() (ll api.LinkedList, v runtime.Value, err error) {
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

func (l List) isNull() bool {
	return !l.v.HasList()
}

func listToSlice(l List, sizeHint int) []runtime.Value {
	slice := make([]runtime.Value, 0, sizeHint)
	runtime.ForEach(l, func(item runtime.Value) bool {
		slice = append(slice, item)
		return false
	})
	return slice
}

func listCons(a capnp.Arena, v runtime.Value, tail List) (l List, err error) {
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

	var cnt int = 1
	if !tail.isNull() {
		cnt = tail.Count() + 1
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
