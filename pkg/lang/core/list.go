package core

import (
	"fmt"

	"github.com/wetware/ww/internal/mem"
	ww "github.com/wetware/ww/pkg"
	memutil "github.com/wetware/ww/pkg/util/mem"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	// EmptyList is a zero-value List
	EmptyList EmptyPersistentList

	emptyList mem.Any

	_ List = (*DeepPersistentList)(nil)
	_ List = (*EmptyPersistentList)(nil)
)

func init() {
	var err error
	if emptyList, err = memutil.Alloc(capnp.SingleSegment(nil)); err != nil {
		panic(err)
	}
}

// List is a persistent, singly-linked list with fast insertions/pops to its head.
type List interface {
	Seq
	Cons(any ww.Any) (List, error)
}

// NewList returns a new list containing given values.
func NewList(a capnp.Arena, items ...ww.Any) (l List, err error) {
	l = EmptyList

	var head ww.Any
	for len(items) > 0 {
		head, items = items[len(items)-1], items[:len(items)-1]
		if l, err = l.Cons(head); err != nil {
			break
		}
	}

	return
}

func asList(any mem.Any) (List, error) {
	list, err := any.List()
	if err != nil {
		return nil, err
	}

	switch list.Which() {
	case mem.LinkedList_Which_empty:
		return EmptyList, nil

	case mem.LinkedList_Which_head:
		return PersistentHeadList{any}, nil

	case mem.LinkedList_Which_packedConsCell:
		return PackedPersistentList{any}, nil

	case mem.LinkedList_Which_consCell:
		return DeepPersistentList{any}, nil
	}

	return nil, fmt.Errorf("%w: invalid type flag '%d' for list",
		ErrMemory,
		list.Which())
}

// EmptyPersistentList is the zero-value list.
type EmptyPersistentList struct{}

// Value returns the memory value.
func (EmptyPersistentList) Value() mem.Any { return emptyList }

// Count the number of items in the list.
func (EmptyPersistentList) Count() (int, error) { return 0, nil }

// Cons appends a value to the head.
func (EmptyPersistentList) Cons(item ww.Any) (List, error) {
	return EmptyList.cons(capnp.SingleSegment(nil), item)
}

func (EmptyPersistentList) cons(a capnp.Arena, item ww.Any) (PersistentHeadList, error) {
	return newPersistentHeadList(a, item)
}

// First returns the first item in the sequence.
func (EmptyPersistentList) First() (ww.Any, error) { return nil, nil }

// Next returns the tail of the sequence (i.e, sequence after
// excluding the head). Returns nil, nil if it has no tail.
func (EmptyPersistentList) Next() (Seq, error) { return nil, nil }

// Conj returns a new sequence with given items conjoined.
func (EmptyPersistentList) Conj(items ...ww.Any) (Container, error) {
	return EmptyList.conj(capnp.SingleSegment(nil), items)
}

func (EmptyPersistentList) conj(a capnp.Arena, items []ww.Any) (List, error) {
	if len(items) == 0 {
		return EmptyList, nil
	}

	l, err := EmptyList.cons(a, items[0])
	if err != nil || len(items) == 1 {
		return l, err
	}

	list, err := l.Any.List()
	if err != nil {
		return nil, err
	}

	return l.conj(capnp.SingleSegment(nil), list, items[1:])
}

// PersistentHeadList is a list without a tail.
type PersistentHeadList struct{ mem.Any }

func newPersistentHeadList(a capnp.Arena, item ww.Any) (PersistentHeadList, error) {
	any, err := memutil.Alloc(a)
	if err != nil {
		return PersistentHeadList{}, err
	}

	list, err := any.NewList()
	if err != nil {
		return PersistentHeadList{}, err
	}

	err = list.SetHead(item.Value())
	return PersistentHeadList{any}, err
}

// Value returns the memory value.
func (l PersistentHeadList) Value() mem.Any { return l.Any }

// Count the number of items in the list.
func (PersistentHeadList) Count() (int, error) { return 1, nil }

// First returns the first item in the sequence.
func (l PersistentHeadList) First() (ww.Any, error) {
	list, err := l.Any.List()
	if err != nil {
		return nil, err
	}

	head, err := list.Head()
	if err != nil {
		return nil, err
	}

	return AsAny(head)
}

// Next returns the tail of the sequence (i.e, sequence after
// excluding the head). Returns nil, nil.
func (PersistentHeadList) Next() (Seq, error) { return nil, nil }

// Cons appends a value to the head.
func (l PersistentHeadList) Cons(item ww.Any) (List, error) {
	list, err := l.List()
	if err != nil {
		return nil, err
	}

	return l.cons(capnp.SingleSegment(nil), list, item)
}

func (l PersistentHeadList) cons(a capnp.Arena, list mem.LinkedList, item ww.Any) (PackedPersistentList, error) {
	any, err := memutil.Alloc(a)
	if err != nil {
		return PackedPersistentList{}, err
	}

	next, err := any.NewList()
	if err != nil {
		return PackedPersistentList{}, err
	}

	// Cons-ing a shallow list always produces a deep list with a shallow tail.
	cell, err := next.NewPackedConsCell()
	if err != nil {
		return PackedPersistentList{}, err
	}

	tail, err := list.Head()
	if err != nil {
		return PackedPersistentList{}, err
	}

	if err = cell.SetTail(tail); err != nil {
		return PackedPersistentList{}, err
	}

	err = cell.SetHead(item.Value())
	return PackedPersistentList{any}, err
}

// Conj returns a new sequence with given items conjoined.
func (l PersistentHeadList) Conj(items ...ww.Any) (Container, error) {
	list, err := l.List()
	if err != nil {
		return nil, err
	}

	return l.conj(capnp.SingleSegment(nil), list, items)
}

func (l PersistentHeadList) conj(a capnp.Arena, list mem.LinkedList, items []ww.Any) (List, error) {
	if len(items) == 0 {
		return l, nil
	}

	ret, err := l.cons(a, list, items[0])
	if err != nil || len(items) == 1 {
		return ret, err
	}

	if list, err = ret.Any.List(); err != nil {
		return nil, err
	}

	return ret.conj(capnp.SingleSegment(nil), list, items[1:])
}

// PackedPersistentList is a compact encoding of a length-2 persistent list
// with O(n) operations at the head and tail.
type PackedPersistentList struct{ mem.Any }

// Value returns the memory value
func (l PackedPersistentList) Value() mem.Any { return l.Any }

// Count returns the number of items in the list.
func (l PackedPersistentList) Count() (int, error) { return 2, nil }

// Conj returns a new list with all the items added at the head of the list.
func (l PackedPersistentList) Conj(items ...ww.Any) (Container, error) {
	list, err := l.List()
	if err != nil {
		return nil, err
	}

	return l.conj(capnp.SingleSegment(nil), list, items)
}

func (l PackedPersistentList) conj(a capnp.Arena, list mem.LinkedList, items []ww.Any) (List, error) {
	if len(items) == 0 {
		return l, nil
	}

	res, err := l.cons(a, list, items[0])
	if err != nil || len(items) == 1 {
		return res, err
	}

	if list, err = res.List(); err != nil {
		return nil, err
	}

	return res.conj(capnp.SingleSegment(nil), list, items[1:])
}

// Cons returns a new list with the item added at the head of the list.
func (l PackedPersistentList) Cons(item ww.Any) (List, error) {
	list, err := l.List()
	if err != nil {
		return nil, err
	}

	return l.cons(capnp.SingleSegment(nil), list, item)
}

func (l PackedPersistentList) cons(a capnp.Arena, list mem.LinkedList, item ww.Any) (DeepPersistentList, error) {
	any, err := memutil.Alloc(a)
	if err != nil {
		return DeepPersistentList{}, err
	}

	next, err := any.NewList()
	if err != nil {
		return DeepPersistentList{}, err
	}

	cell, err := next.NewConsCell()
	if err != nil {
		return DeepPersistentList{}, err
	}

	if err = cell.SetHead(item.Value()); err != nil {
		return DeepPersistentList{}, err
	}

	err = cell.SetTail(l.Any)
	return DeepPersistentList{any}, err
}

// First returns the head or first item of the list.
func (l PackedPersistentList) First() (ww.Any, error) {
	list, err := l.List()
	if err != nil {
		return nil, err
	}

	cell, err := list.PackedConsCell()
	if err != nil {
		return nil, err
	}

	head, err := cell.Head()
	if err != nil {
		return nil, err
	}

	return AsAny(head)
}

// Next returns the tail of the list.
func (l PackedPersistentList) Next() (Seq, error) {
	list, err := l.List()
	if err != nil {
		return nil, err
	}

	cell, err := list.PackedConsCell()
	if err != nil {
		return nil, err
	}

	tail, err := cell.Tail()
	if err != nil {
		return nil, err
	}

	return newPersistentHeadList(capnp.SingleSegment(nil), item(tail))
}

// DeepPersistentList is an immutable, singley-linked list with fast
// head operations.
type DeepPersistentList struct{ mem.Any }

// Value returns the memory value
func (l DeepPersistentList) Value() mem.Any { return l.Any }

// Count returns the number of the list.
func (l DeepPersistentList) Count() (int, error) {
	list, err := l.Any.List()
	if err != nil {
		return 0, err
	}

	off, err := l.offset(list)
	return int(off + 3), err
}

func (l DeepPersistentList) offset(list mem.LinkedList) (uint32, error) {
	cell, err := list.ConsCell()
	return cell.Offset(), err
}

// Conj returns a new list with all the items added at the head of the list.
func (l DeepPersistentList) Conj(items ...ww.Any) (Container, error) {
	list, err := l.List()
	if err != nil {
		return DeepPersistentList{}, err
	}

	return l.conj(capnp.SingleSegment(nil), list, items)
}

func (l DeepPersistentList) conj(a capnp.Arena, list mem.LinkedList, items []ww.Any) (DeepPersistentList, error) {
	if len(items) == 0 {
		return l, nil
	}

	ret, err := l.cons(a, list, items[0])
	if err != nil || len(items) == 1 {
		return ret, err
	}

	if list, err = ret.List(); err != nil {
		return DeepPersistentList{}, err
	}

	return ret.conj(capnp.SingleSegment(nil), list, items[1:])
}

// Cons returns a new list with the item added at the head of the list.
func (l DeepPersistentList) Cons(item ww.Any) (List, error) {
	list, err := l.List()
	if err != nil {
		return nil, err
	}

	return l.cons(capnp.SingleSegment(nil), list, item)
}

func (l DeepPersistentList) cons(a capnp.Arena, list mem.LinkedList, item ww.Any) (DeepPersistentList, error) {
	any, err := memutil.Alloc(a)
	if err != nil {
		return DeepPersistentList{}, err
	}

	next, err := any.NewList()
	if err != nil {
		return DeepPersistentList{}, err
	}

	cell, err := next.NewConsCell()
	if err != nil {
		return DeepPersistentList{}, err
	}

	if err = cell.SetHead(item.Value()); err != nil {
		return DeepPersistentList{}, err
	}

	if err = cell.SetTail(l.Any); err != nil {
		return DeepPersistentList{}, err
	}

	off, err := l.offset(list)
	if err != nil {
		return DeepPersistentList{}, err
	}
	cell.SetOffset(off + 1)

	return DeepPersistentList{any}, err
}

// First returns the head or first item of the list.
func (l DeepPersistentList) First() (ww.Any, error) {
	list, err := l.Any.List()
	if err != nil {
		return nil, err
	}

	cell, err := list.ConsCell()
	if err != nil {
		return nil, err
	}

	head, err := cell.Head()
	if err != nil {
		return nil, err
	}

	return AsAny(head)
}

// Next returns the tail of the list.
func (l DeepPersistentList) Next() (Seq, error) {
	list, err := l.Any.List()
	if err != nil {
		return nil, err
	}

	cell, err := list.ConsCell()
	if err != nil {
		return nil, err
	}

	tail, err := cell.Tail()
	if err != nil {
		return nil, err
	}

	item, err := asList(tail)
	if err != nil {
		return nil, err
	}

	if seq, ok := item.(Seq); ok {
		return seq, nil
	}

	return nil, fmt.Errorf("%w: deep list must have sequence tail", ErrMemory)
}
