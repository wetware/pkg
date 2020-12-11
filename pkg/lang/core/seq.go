package core

import (
	"strings"

	ww "github.com/wetware/ww/pkg"
	capnp "zombiezen.com/go/capnproto2"
)

// Seq represents a sequence of values.
type Seq interface {
	ww.Any

	// Count returns the number of items in the sequence.
	Count() (int, error)

	// First returns the first item in the sequence.
	First() (ww.Any, error)

	// Next returns the tail of the sequence (i.e, sequence after
	// excluding the head). Returns nil, nil if it has no tail.
	Next() (Seq, error)

	// Conj returns a new sequence with given items conjoined.
	Conj(...ww.Any) (Container, error)
}

// Seqable types can be represented as a sequence.
type Seqable interface {
	// Return a sequence representation of the underlying type.
	Seq() (Seq, error)
}

// Cons .
func Cons(a capnp.Arena, head ww.Any, tail Seq) (List, error) {
	tailcnt, err := tail.Count()
	if err != nil {
		return nil, err
	}

	if tailcnt == 0 {
		return EmptyList.cons(a, head)
	}

	list, err := tail.Value().List()
	if err != nil {
		return nil, err
	}

	switch tailcnt {
	case 1:
		return PersistentHeadList{tail.Value()}.cons(a, list, head)

	case 2:
		return PackedPersistentList{tail.Value()}.cons(a, list, head)

	default:
		return DeepPersistentList{tail.Value()}.cons(a, list, head)
	}
}

// ToSlice converts the given sequence into a slice.
func ToSlice(seq Seq) ([]ww.Any, error) {
	cnt, err := seq.Count()
	if err != nil || cnt == 0 {
		return nil, err
	}

	sl := make([]ww.Any, 0, cnt)
	err = ForEach(seq, func(item ww.Any) (bool, error) {
		sl = append(sl, item)
		return false, nil
	})
	return sl, err
}

// ForEach reads from the sequence and calls the given function for each item.
// Function can return true to stop the iteration.
func ForEach(seq Seq, call func(item ww.Any) (bool, error)) (err error) {
	var v ww.Any
	var done bool
	for seq != nil {
		if v, err = seq.First(); err != nil || v == nil {
			break
		}

		if done, err = call(v); err != nil || done {
			break
		}

		if seq, err = seq.Next(); err != nil {
			break
		}
	}

	return err
}

// SeqString returns a string representation for the sequence with given prefix
// suffix and separator.
func SeqString(seq Seq, begin, end, sep string) (string, error) {
	var b strings.Builder
	b.WriteString(begin)

	err := ForEach(seq, func(item ww.Any) (bool, error) {
		str, err := Render(item)
		if err == nil {
			b.WriteString(str)
			b.WriteString(sep)
		}

		return false, err
	})

	return strings.TrimRight(b.String(), sep) + end, err
}

func renderSeq(seq Seq) (string, error) {
	var b strings.Builder
	b.WriteRune('(')

	err := ForEach(seq, func(item ww.Any) (_ bool, err error) {
		if b.Len() > 1 {
			b.WriteRune(' ')
		}

		var s string
		if s, err = Render(item); err == nil {
			b.WriteString(s)
		}

		return
	})

	b.WriteRune(')')
	return b.String(), err
}
