package core

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/wetware/ww/internal/mem"
	ww "github.com/wetware/ww/pkg"
	memutil "github.com/wetware/ww/pkg/util/mem"
	capnp "zombiezen.com/go/capnproto2"
)

/*
	vector.go contains a persistent bit-partitioned vector implementation.

	TODO(performance):  implement transients.

	TODO(performance):  investigate RRB tree for fast concats/prepends
						http://infoscience.epfl.ch/record/169879/files/RMTrees.pdf
*/

const (
	bits  = 5 // number of bits needed to represent the range (0 32].
	width = 32
	mask  = width - 1 // 0x1f
)

var (
	// ErrInvalidVectorNode is returned when a node in the vector trie is invalid.
	ErrInvalidVectorNode = errors.New("invalid VectorNode")

	// EmptyVector is the zero-value vector.
	EmptyVector EmptyPersistentVector

	_ Vector = (*EmptyPersistentVector)(nil)
	_ Vector = (*ShallowPersistentVector)(nil)
	_ Vector = (*DeepPersistentVector)(nil)

	emptyVector    mem.Any
	emptyVectorSeq chunkedSeq
)

func init() {
	any, err := memutil.Alloc(capnp.SingleSegment(nil))
	if err != nil {
		panic(err)
	}

	vec, err := any.NewVector()
	if err != nil {
		panic(err)
	}

	emptyVector = any

	emptyVectorSeq, err = newChunkedSeq(capnp.SingleSegment(nil), vec, 0, 0)
	if err != nil {
		panic(err)
	}
}

// Vector is a persistent, ordered collection of values with fast random lookups and
// insertions.
type Vector interface {
	ww.Any
	Seq() (Seq, error)
	Count() (int, error)
	Conj(...ww.Any) (Container, error)
	EntryAt(i int) (ww.Any, error)
	Assoc(i int, val ww.Any) (Vector, error)
	Pop() (Vector, error)
	Cons(ww.Any) (Vector, error)
}

// NewVector creates a vector containing the supplied values.
func NewVector(a capnp.Arena, items ...ww.Any) (Vector, error) {
	return EmptyVector.conj(items)
}

// EmptyPersistentVector is the zero-value persistent vector.
type EmptyPersistentVector struct{}

// Value returns the memory value.
func (EmptyPersistentVector) Value() mem.Any { return emptyVector }

// Count always returns 0 and a nil error.
func (EmptyPersistentVector) Count() (int, error) { return 0, nil }

// Invoke is equivalent to `EntryAt`.
func (EmptyPersistentVector) Invoke(args ...ww.Any) (ww.Any, error) {
	if nargs := len(args); nargs != 1 {
		return nil, fmt.Errorf("%w: got %d, want at-least 1", ErrArity, nargs)
	}

	switch idx := args[0]; idx.Value().Which() {
	case mem.Any_Which_i64, mem.Any_Which_bigInt:
		return nil, ErrIndexOutOfBounds

	default:
		return nil, fmt.Errorf("%s is not an integer type", idx.Value().Which())
	}
}

// Render the vector in a human-readable format.
func (EmptyPersistentVector) Render() (string, error) { return "[]", nil }

// EntryAt returns ErrIndexOutOfBounds
func (EmptyPersistentVector) EntryAt(int) (ww.Any, error) { return nil, ErrIndexOutOfBounds }

// Conj returns a new vector with items appended.
func (EmptyPersistentVector) Conj(items ...ww.Any) (Container, error) {
	if len(items) == 0 {
		return EmptyVector, nil
	}

	return EmptyVector.conj(items)
}

// Assoc returns a new vector with the value at given index updated.
// Returns error if the index is out of range.
func (EmptyPersistentVector) Assoc(i int, val ww.Any) (Vector, error) {
	if i != 0 {
		return nil, ErrIndexOutOfBounds
	}

	return EmptyVector.cons(val)
}

func (EmptyPersistentVector) conj(items []ww.Any) (Vector, error) {
	if len(items) == 0 {
		return EmptyVector, nil
	}

	if len(items) <= width {
		return newShallowPersistentVector(capnp.SingleSegment(nil), items...)
	}

	any, err := memutil.Alloc(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	vec, err := any.NewVector()
	if err != nil {
		return nil, err
	}

	vec.SetCount(width) // 32
	vec.SetShift(bits)  // 5

	tail, err := vec.NewTail(width)
	if err != nil {
		return nil, err
	}

	for i, any := range items[:width] {
		if err = tail.Set(i, any.Value()); err != nil {
			return nil, err
		}
	}

	root, err := vec.NewRoot()
	if err != nil {
		return nil, err
	}

	if _, err = root.NewBranches(width); err != nil {
		return nil, err
	}

	return (DeepPersistentVector{any}).conj(items[width:])
}

// Cons appends to the end of the vector
func (EmptyPersistentVector) Cons(item ww.Any) (Vector, error) {
	return (EmptyVector).cons(item)
}

func (EmptyPersistentVector) cons(any ww.Any) (ShallowPersistentVector, error) {
	return newShallowPersistentVector(capnp.SingleSegment(nil), any)
}

// Pop returns ErrIllegalState
func (EmptyPersistentVector) Pop() (Vector, error) {
	return nil, fmt.Errorf("%w: cannot pop from empty vector", ErrIllegalState)
}

// Seq returns an empty
func (EmptyPersistentVector) Seq() (Seq, error) { return emptyVectorSeq, nil }

// ShallowPersistentVector is a compact vector that stores up to 32 values.
type ShallowPersistentVector struct{ mem.Any }

func newShallowPersistentVector(a capnp.Arena, items ...ww.Any) (ShallowPersistentVector, error) {
	if len(items) == 0 {
		return ShallowPersistentVector{}, fmt.Errorf("%w: ShallowPersistentVector must not be empty", ErrIllegalState)

	}

	if len(items) > width {
		return ShallowPersistentVector{}, fmt.Errorf("%w: cannot allocate %d items to ShallowPersistentVector (max %d)",
			ErrIllegalState,
			len(items),
			width)
	}

	val, err := memutil.Alloc(a)
	if err != nil {
		return ShallowPersistentVector{}, err
	}

	vec, err := val.NewVector()
	if err != nil {
		return ShallowPersistentVector{}, err
	}

	vec.SetCount(uint32(len(items)))

	tail, err := vec.NewTail(width)
	if err != nil {
		return ShallowPersistentVector{}, err
	}

	for i, any := range items {
		if err = tail.Set(i, any.Value()); err != nil {
			break
		}
	}

	return ShallowPersistentVector{val}, err
}

// Value returns the memory value
func (v ShallowPersistentVector) Value() mem.Any { return v.Any }

// Count returns the tail length
func (v ShallowPersistentVector) Count() (int, error) {
	vec, err := v.Any.Vector()
	return int(vec.Count()), err
}

// Invoke is equivalent to `EntryAt`.
func (ShallowPersistentVector) Invoke(args ...ww.Any) (ww.Any, error) {
	if nargs := len(args); nargs != 1 {
		return nil, fmt.Errorf("%w: got %d, want at-least 1", ErrArity, nargs)
	}

	switch idx := args[0]; idx.Value().Which() {
	case mem.Any_Which_i64, mem.Any_Which_bigInt:
		return nil, ErrIndexOutOfBounds

	default:
		return nil, fmt.Errorf("%s is not an integer type", idx.Value().Which())
	}
}

// Render the vector in a human-readable format.
func (v ShallowPersistentVector) Render() (string, error) {
	vec, err := v.Any.Vector()
	if err != nil {
		return "", err
	}

	tail, err := vec.Tail()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteRune('[')

	cnt := int(vec.Count())
	for i := 0; i < cnt; i++ {
		item, err := AsAny(tail.At(i))
		if err != nil {
			return "", err
		}

		s, err := Render(item)
		if err != nil {
			return "", err
		}

		b.WriteString(s)

		if i < cnt-1 {
			b.WriteRune(' ')
		}
	}

	b.WriteRune(']')
	return b.String(), nil
}

// Assoc returns a new vector with the value at given index updated.
// Returns error if the index is out of range.
func (v ShallowPersistentVector) Assoc(i int, item ww.Any) (Vector, error) {
	vec, err := v.Any.Vector()
	if err != nil {
		return nil, err
	}

	cnt := int(vec.Count())

	// update?
	if i >= 0 && i < cnt {
		return v.update(vec, cnt, i, item)
	}

	// append?
	if i == cnt {
		return v.cons(vec, item)
	}

	return nil, ErrIndexOutOfBounds
}

// EntryAt returns ErrIndexOutOfBounds
func (v ShallowPersistentVector) EntryAt(i int) (ww.Any, error) {
	if i > 31 || i < 0 {
		return nil, ErrIndexOutOfBounds
	}

	vec, err := v.Any.Vector()
	if err != nil {
		return nil, err
	}

	tail, err := vec.Tail()
	if err != nil {
		return nil, err
	}

	return AsAny(tail.At(i))
}

// Conj returns a new vector with items appended.
func (v ShallowPersistentVector) Conj(items ...ww.Any) (Container, error) {

	return v.conj(items)
}

func (v ShallowPersistentVector) conj(items []ww.Any) (Vector, error) {
	if len(items) == 0 {
		return v, nil
	}

	vec, err := v.Any.Vector()
	if err != nil {
		return nil, err
	}

	cnt := int(vec.Count())

	// result fits in shallow vector?
	if cnt+len(items) <= width {
		tail, err := vec.Tail()
		if err != nil {
			return nil, err
		}

		ts := tailSlice(cnt, tail)
		for _, val := range items {
			ts = append(ts, val)
		}

		return newShallowPersistentVector(capnp.SingleSegment(nil), ts...)
	}

	// deep vector is needed
	newtail, err := v.cloneTail(capnp.SingleSegment(nil), vec)
	if err != nil {
		return nil, err
	}

	if cnt < width {
		offset := width - cnt // number of free slots in tail
		for i := 0; i < offset; i++ {
			if err = newtail.Set(cnt+i, items[i].Value()); err != nil {
				return nil, err
			}
		}

		items = items[offset:]
	}

	return newDeepPersistentVector(capnp.SingleSegment(nil),
		newtail,
		items...)
}

func (v ShallowPersistentVector) cloneTail(a capnp.Arena, vec mem.Vector) (newtail mem.Any_List, err error) {
	var tail mem.Any_List
	if tail, err = vec.Tail(); err != nil {
		return
	}

	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	if newtail, err = mem.NewAny_List(seg, width); err == nil {
		for i := 0; i < int(vec.Count()); i++ {
			if err = newtail.Set(i, tail.At(i)); err != nil {
				break
			}
		}
	}

	return
}

// Cons appends to the end of the vector.
func (v ShallowPersistentVector) Cons(item ww.Any) (Vector, error) {
	vec, err := v.Any.Vector()
	if err != nil {
		return nil, err
	}

	return v.cons(vec, item)
}

func (v ShallowPersistentVector) cons(vec mem.Vector, any ww.Any) (Vector, error) {
	if cnt := int(vec.Count()); cnt < width {
		return v.shallowCons(vec, cnt, any)
	}

	return v.deepCons(vec, any)
}

func (v ShallowPersistentVector) shallowCons(vec mem.Vector, cnt int, item ww.Any) (ShallowPersistentVector, error) {
	tail, err := vec.Tail()
	if err != nil {
		return ShallowPersistentVector{}, err
	}

	any, err := memutil.Alloc(capnp.SingleSegment(nil))
	if err != nil {
		return ShallowPersistentVector{}, err
	}

	if vec, err = any.NewVector(); err != nil {
		return ShallowPersistentVector{}, err
	}

	vec.SetCount(uint32(cnt + 1))

	newTail, err := vec.NewTail(width)
	if err != nil {
		return ShallowPersistentVector{}, err
	}

	for i := 0; i < cnt; i++ {
		if err = newTail.Set(i, tail.At(i)); err != nil {
			break
		}
	}

	return ShallowPersistentVector{any}, newTail.Set(cnt, item.Value())
}

func (v ShallowPersistentVector) deepCons(vec mem.Vector, any ww.Any) (DeepPersistentVector, error) {
	tail, err := vec.Tail()
	if err != nil {
		return DeepPersistentVector{}, err
	}

	return newDeepPersistentVector(capnp.SingleSegment(nil),
		tail,
		any)
}

func (v ShallowPersistentVector) update(vec mem.Vector, cnt, idx int, item ww.Any) (ShallowPersistentVector, error) {
	tail, err := vec.Tail()
	if err != nil {
		return ShallowPersistentVector{}, err
	}

	any, err := memutil.Alloc(capnp.SingleSegment(nil))
	if err != nil {
		return ShallowPersistentVector{}, err
	}

	newVec, err := any.NewVector()
	if err != nil {
		return ShallowPersistentVector{}, err
	}

	newVec.SetCount(uint32(cnt))

	newTail, err := vec.NewTail(width)
	if err != nil {
		return ShallowPersistentVector{}, err
	}

	for i := 0; i < int(cnt); i++ {
		if i == idx {
			err = newTail.Set(i, item.Value())
		} else {
			err = newTail.Set(i, tail.At(i))
		}

		if err != nil {
			break
		}
	}

	return ShallowPersistentVector{any}, err
}

// Pop returns ErrIllegalState
func (v ShallowPersistentVector) Pop() (Vector, error) {
	vec, err := v.Any.Vector()
	if err != nil {
		return nil, err
	}

	cnt := vec.Count()
	if cnt == 1 {
		return EmptyVector, nil
	}

	tail, err := vec.Tail()
	if err != nil {
		return nil, err
	}

	any, err := memutil.Alloc(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	if vec, err = any.NewVector(); err != nil {
		return nil, err
	}

	vec.SetCount(cnt - 1)

	newTail, err := vec.NewTail(width)
	if err != nil {
		return nil, err
	}

	for i := 0; i < int(cnt-1); i++ {
		if err = newTail.Set(i, tail.At(i)); err != nil {
			break
		}
	}

	return ShallowPersistentVector{any}, err
}

// Seq returns a sequence that iterates over the vector
func (v ShallowPersistentVector) Seq() (Seq, error) {
	vec, err := v.Any.Vector()
	if err != nil {
		return nil, err
	}

	any, err := memutil.Alloc(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	seq, err := any.NewVectorSeq()
	if err != nil {
		return nil, err
	}

	if err = seq.SetVector(vec); err != nil {
		return nil, err
	}

	return chunkedSeq{any}, nil
}

// DeepPersistentVector is a persistent, immutable vector.
type DeepPersistentVector struct{ mem.Any }

// N.B.:  tail MUST be of length 32 and fully-populated.
func newDeepPersistentVector(a capnp.Arena, tail mem.Any_List, items ...ww.Any) (DeepPersistentVector, error) {
	any, err := memutil.Alloc(a)
	if err != nil {
		return DeepPersistentVector{}, err
	}

	vec, err := any.NewVector()
	if err != nil {
		return DeepPersistentVector{}, err
	}

	if err = vec.SetTail(tail); err != nil {
		return DeepPersistentVector{}, err
	}

	vec.SetCount(width) // 32
	vec.SetShift(bits)  // 5

	root, err := vec.NewRoot()
	if err != nil {
		return DeepPersistentVector{}, err
	}

	if _, err = root.NewBranches(width); err != nil {
		return DeepPersistentVector{}, err
	}

	return (DeepPersistentVector{any}).conj(items)
}

// Value returns the memory value
func (v DeepPersistentVector) Value() mem.Any { return v.Any }

// Invoke is equivalent to `EntryAt`.
func (v DeepPersistentVector) Invoke(args ...ww.Any) (ww.Any, error) {
	if nargs := len(args); nargs != 1 {
		return nil, fmt.Errorf("%w: got %d, want at-least 1", ErrArity, nargs)
	}

	switch idx := args[0]; idx.Value().Which() {
	case mem.Any_Which_i64:
		return v.EntryAt(int(idx.Value().I64()))
	case mem.Any_Which_bigInt:
		// TODO(performance):  can we use unsafe.Pointer here?
		if bi := idx.(BigInt).BigInt(); bi.IsInt64() && bi.Int64() <= math.MaxUint32 {
			return v.EntryAt(int(bi.Int64()))
		}

		return nil, ErrIndexOutOfBounds

	default:
		return nil, fmt.Errorf("%s is not an integer type", idx.Value().Which())
	}
}

// Render the vector in a human-readable format.
func (v DeepPersistentVector) Render() (string, error) {
	cnt, err := v.Count()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteRune('[')

	for i := 0; i < cnt; i++ {
		val, err := v.EntryAt(i)
		if err != nil {
			return "", fmt.Errorf("%w: index %d", err, i)
		}

		s, err := Render(val)
		if err != nil {
			return "", fmt.Errorf("%w: index %d", err, i)
		}

		b.WriteString(s)

		if i < cnt-1 {
			b.WriteRune(' ')
		}
	}

	b.WriteRune(']')
	return b.String(), nil
}

// Count returns the number of elements in the vector.
func (v DeepPersistentVector) Count() (cnt int, err error) {
	_, cnt, err = v.count()
	return
}

func (v DeepPersistentVector) count() (vec mem.Vector, cnt int, err error) {
	if vec, err = v.Any.Vector(); err == nil {
		cnt = int(vec.Count())
	}

	return
}

// Conj returns a new vector with items appended.
func (v DeepPersistentVector) Conj(items ...ww.Any) (Container, error) { return v.conj(items) }

func (v DeepPersistentVector) conj(items []ww.Any) (DeepPersistentVector, error) {
	for _, any := range items {
		vec, cnt, err := v.count()
		if err != nil {
			return DeepPersistentVector{}, err
		}

		if v, err = v.cons(vec, cnt, any); err != nil {
			return DeepPersistentVector{}, err
		}
	}

	return v, nil
}

// EntryAt returns the item at given index. Returns error if the index
// is out of range.
func (v DeepPersistentVector) EntryAt(i int) (ww.Any, error) {
	vs, err := v.arrayFor(i)
	if err != nil {
		return nil, err
	}

	return AsAny(vs.At(i & mask))
}

// Assoc returns a new vector with the value at given index updated.
// Returns error if the index is out of range.
func (v DeepPersistentVector) Assoc(i int, item ww.Any) (Vector, error) {
	// https://github.com/clojure/clojure/blob/0b73494c3c855e54b1da591eeb687f24f608f346/src/jvm/clojure/lang/PersistentVector.java#L121

	vec, cnt, err := v.count()
	if err != nil {
		return nil, err
	}

	// update?
	if i >= 0 && i < cnt {
		return v.update(vec, cnt, i, item)
	}

	// append?
	if i == cnt {
		return v.cons(vec, cnt, item)
	}

	return nil, ErrIndexOutOfBounds
}

// Pop returns a new vector without the last item in v
func (v DeepPersistentVector) Pop() (Vector, error) {
	raw, err := v.Any.Vector()
	if err != nil {
		return nil, err
	}

	if vec, err := v.pop(raw); err != ErrIllegalState {
		return vec, err
	}

	return nil, fmt.Errorf("%w: cannot pop from empty vector", ErrIllegalState)
}

// Seq presents the vector as an iterable sequence.
func (v DeepPersistentVector) Seq() (Seq, error) {
	vec, err := v.Any.Vector()
	if err != nil {
		return nil, err
	}

	if vec.Count() == 0 {
		return EmptyList, nil
	}

	return newChunkedSeq(capnp.SingleSegment(nil), vec, 0, 0)
}

func (v DeepPersistentVector) popTail(level, cnt int, n mem.Vector_Node) (ret mem.Vector_Node, ok bool, err error) {
	subidx := ((cnt - 2) >> level) & mask
	if level > 5 {
		var bs mem.Vector_Node_List
		if bs, err = n.Branches(); err != nil {
			return
		}

		var newchild mem.Vector_Node
		switch newchild, ok, err = v.popTail(level-5, cnt, bs.At(subidx)); {
		case err != nil, !ok && subidx == 0:
			return
		}

		if ret, err = cloneBranchNode(capnp.SingleSegment(nil), n, subidx); err != nil {
			return
		}

		if bs, err = ret.Branches(); err != nil {
			return
		}

		if err = bs.Set(subidx, newchild); err != nil {
			return
		}

		ok = true
		return
	} else if subidx == 0 {
		return // null node
	} else {
		// ret.array[subidx] = null;
		if ret, err = cloneNode(capnp.SingleSegment(nil), n, subidx); err != nil {
			return
		}

		ok = true
		return
	}
}

func (v DeepPersistentVector) arrayFor(i int) (mem.Any_List, error) {
	// See:  https://github.com/clojure/clojure/blob/0b73494c3c855e54b1da591eeb687f24f608f346/src/jvm/clojure/lang/PersistentVector.java#L97-L113
	vec, cnt, err := v.count()
	if err == nil {
		if i < 0 || i >= cnt {
			return mem.Any_List{}, ErrIndexOutOfBounds
		}
	}

	return apiVectorArrayFor(vec, int(cnt), i)
}

func (DeepPersistentVector) update(vec mem.Vector, cnt, i int, any ww.Any) (Vector, error) {
	root, err := vec.Root()
	if err != nil {
		return nil, err
	}

	tail, err := vec.Tail()
	if err != nil {
		return nil, err
	}

	// index is in tail?
	if tailoff := vectorTailoff(cnt); i >= tailoff {
		oldtail := tail

		if tail, err = newVectorValueList(capnp.SingleSegment(nil)); err != nil {
			return nil, err
		}

		taillen := cnt - tailoff
		for i := 0; i < taillen; i++ {
			if err = tail.Set(i, oldtail.At(i)); err != nil {
				return nil, err
			}
		}

		if err = tail.Set(i&mask, any.Value()); err != nil {
			return nil, err
		}
	} else {
		if root, err = apiVectorAssoc(int(vec.Shift()), root, i, any.Value()); err != nil {
			return nil, err
		}
	}

	return newVector(capnp.SingleSegment(nil),
		cnt,
		int(vec.Shift()),
		root,
		tail,
	)
}

func (v DeepPersistentVector) shift(vec mem.Vector) (shift int) {
	// EmptyPersistentVector leaves the `shift` field unset in order to achieve
	// better compression.
	if shift = int(vec.Shift()); shift == 0 {
		shift = bits
	}

	return
}

// Cons appends to the end of the vector.
func (v DeepPersistentVector) Cons(item ww.Any) (Vector, error) {
	vec, err := v.Any.Vector()
	if err != nil {
		return nil, err
	}

	return v.cons(vec, int(vec.Count()), item)
}

func (v DeepPersistentVector) cons(vec mem.Vector, cnt int, any ww.Any) (_ DeepPersistentVector, err error) {
	shift := v.shift(vec)

	var root mem.Vector_Node
	if root, err = vec.Root(); err != nil {
		return
	}

	var tail mem.Any_List
	if tail, err = vec.Tail(); err != nil {
		return
	}

	/*
		Fast path; room in tail?
	*/
	if taillen := cnt - vectorTailoff(cnt); taillen < width {
		var newtail mem.Any_List
		if newtail, err = newVectorValueList(capnp.SingleSegment(nil)); err != nil {
			return
		}

		// copy old values to new tail
		for i := 0; i < taillen; i++ {
			if err = newtail.Set(i, tail.At(i)); err != nil {
				return
			}
		}

		// append the new value to the new tail
		if err = newtail.Set(taillen, any.Value()); err != nil {
			return
		}

		return newVector(capnp.SingleSegment(nil),
			cnt+1,
			shift,
			root,
			newtail)
	}

	/*
		Slow path; push to tree
	*/

	var newroot mem.Vector_Node

	// Wrap the tail in a node so that we can push it into the trie.
	var tailnode mem.Vector_Node
	if tailnode, err = v.newLeafNode(capnp.SingleSegment(nil), tail); err != nil {
		return
	}

	// Overflow root?
	if (cnt >> bits) > (1 << shift) {
		if newroot, err = newRootVectorNode(capnp.SingleSegment(nil)); err != nil {
			return
		}

		var array mem.Vector_Node_List
		if array, err = newroot.NewBranches(width); err != nil {
			return
		}

		// first branch points to old root
		if err = array.Set(0, root); err != nil {
			return
		}

		// second branch points to former tail
		var path mem.Vector_Node
		if path, err = v.newPath(shift, tailnode); err != nil {
			return
		}

		if err = array.Set(1, path); err != nil {
			return
		}

		shift += bits
	} else {
		if newroot, err = v.pushTail(shift, cnt, root, tailnode); err != nil {
			return
		}
	}

	// old tail was successfully inserted; create new tail...
	var newtail mem.Any_List
	if newtail, err = newVectorValueList(capnp.SingleSegment(nil)); err != nil {
		return
	}

	// ... and insert new value into the new tail.
	if err = newtail.Set(0, any.Value()); err != nil {
		return
	}

	return newVector(capnp.SingleSegment(nil),
		cnt+1,
		shift,
		newroot,
		newtail)
}

// vs is always the old tail, which is now being pushed into the trie.
func (DeepPersistentVector) newLeafNode(a capnp.Arena, vs mem.Any_List) (n mem.Vector_Node, err error) {
	if n, err = newRootVectorNode(a); err == nil {
		err = n.SetValues(vs)
	}

	return
}

func apiVectorAssoc(level int, n mem.Vector_Node, i int, val mem.Any) (ret mem.Vector_Node, err error) {
	if ret, err = cloneNode(capnp.SingleSegment(nil), n, width); err != nil {
		return
	}

	// is leaf?
	if level == 0 {
		var vs mem.Any_List
		if vs, err = ret.Values(); err == nil {
			err = vs.Set(i&mask, val)
		}

		return
	}

	// else assoc branch

	var bs mem.Vector_Node_List
	if bs, err = n.Branches(); err != nil {
		return
	}

	subidx := (i >> level) & mask
	if n, err = apiVectorAssoc(level-bits, bs.At(subidx), i, val); err != nil {
		return
	}

	if bs, err = ret.Branches(); err == nil {
		err = bs.Set(subidx, n)
	}

	return

}

// number of items in the actual trie (i.e. not in the tail)
func vectorTailoff(cnt int) int {
	if cnt < width {
		return 0
	}

	return ((cnt - 1) >> bits) << bits
}

func (v DeepPersistentVector) pop(vec mem.Vector) (_ Vector, err error) {
	cnt := int(vec.Count())
	switch cnt {
	case 0:
		return nil, ErrIllegalState
	case 1:
		return EmptyVector, nil
	}

	var root mem.Vector_Node
	if root, err = vec.Root(); err != nil {
		return
	}

	/*
		Fast path.  There's more than one item in the tail, so we won't
		have to pop the old tail and dig up a node from the trie.
	*/
	var newtail mem.Any_List
	if taillen := cnt - vectorTailoff(cnt); taillen > 1 {
		var tail mem.Any_List
		if tail, err = vec.Tail(); err != nil {
			return
		}

		// result fits in shallow vector?
		if cnt-1 <= width {
			return newShallowPersistentVector(capnp.SingleSegment(nil),
				tailSlice(cnt, tail)...)
		}

		if newtail, err = newVectorValueList(capnp.SingleSegment(nil)); err != nil {
			return
		}

		for i := 0; i < taillen-1; i++ {
			if err = newtail.Set(i, tail.At(i)); err != nil {
				return
			}
		}

		return newVector(capnp.SingleSegment(nil),
			cnt-1,
			int(vec.Shift()),
			root,
			newtail)
	}

	/*
		Slow path.  There's a single item in the tail, so we'll have to
		pop the old tail and dig a new one out of the trie.
	*/

	if newtail, err = v.arrayFor(cnt - 2); err != nil {
		return
	}

	if cnt-1 <= width {
		return newShallowPersistentVector(capnp.SingleSegment(nil),
			tailSlice(cnt-1, newtail)...)
	}

	// vec.Shift() >= bits since EmptyPersistentVector.Pop() aborts with an error.
	shift := int(vec.Shift())

	var ok bool
	var newroot mem.Vector_Node
	if newroot, ok, err = v.popTail(shift, cnt, root); err != nil {
		return
	}

	// null node?
	if !ok {
		// 	newroot = EMPTY_NODE;
		if newroot, err = newVectorNodeWithBranches(capnp.SingleSegment(nil)); err != nil {
			return
		}
	}

	var bs mem.Vector_Node_List
	if bs, err = newroot.Branches(); err != nil {
		return
	}

	if shift > bits && nullNode(bs.At(1)) {
		newroot = bs.At(0)
		shift -= bits
	}

	return newVector(capnp.SingleSegment(nil),
		cnt-1,
		shift,
		newroot,
		newtail)
}

func (v DeepPersistentVector) newPath(level int, node mem.Vector_Node) (ret mem.Vector_Node, err error) {
	if level == 0 {
		return node, nil
	}

	if ret, err = newRootVectorNode(capnp.SingleSegment(nil)); err != nil {
		return
	}

	var array mem.Vector_Node_List
	/*
		TODO(optimization)
		Right now we allocate fixed-size branches.  This reduces the number of allocations
		when building large vectors, but wastes a bit of space on the wire.  Investigate
		whether we can efficiently grow branches.

		Note that this is harder than it seems.  In pushTail, for example, We can't mutate
		an existing node, else we lose immutability guarantees.  We therefore have to create
		a new node each time the branch array grows, which is expensive.  It is likely that
		we will need to resort some kind of pooling strategy to offset the cost of allocation,
		but it might end up being more performant to waste a bit of space in branch arrays.

		In any case:  resist the urge to optimize this before solid benchmarks are in place.
	*/
	if array, err = ret.NewBranches(width); err != nil {
		return
	}

	var path mem.Vector_Node
	if path, err = v.newPath(level-bits, node); err == nil {
		err = array.Set(0, path)
	}

	return
}

func (v DeepPersistentVector) pushTail(level, cnt int, parent, tailnode mem.Vector_Node) (_ mem.Vector_Node, err error) {
	// if parent is leaf => insert node,
	//   else does it map to an existing child? => nodeToInsert = pushNode one more level
	//   else => alloc new path
	//
	// return nodeToInsert placed in parent

	var nodeToInsert mem.Vector_Node
	subidx := ((cnt - 1) >> level) & mask

	// parent is leaf?
	if level == bits {
		nodeToInsert = tailnode
	} else {
		var child mem.Vector_Node
		if child, err = getChild(parent, subidx); err != nil {
			return
		}

		if nodeNotNull(child) {
			nodeToInsert, err = v.pushTail(level-bits, cnt, child, tailnode)
		} else {
			nodeToInsert, err = v.newPath(level-bits, tailnode)
		}

		if err != nil {
			return
		}
	}

	var bs mem.Vector_Node_List
	if bs, err = parent.Branches(); err == nil {
		err = bs.Set(subidx, nodeToInsert)
	}

	return parent, err
}

func nullNode(n mem.Vector_Node) bool {
	return !n.HasBranches() && !n.HasValues()
}

func nodeNotNull(n mem.Vector_Node) bool {
	return n.HasBranches() || n.HasValues()
}

func getChild(p mem.Vector_Node, i int) (mem.Vector_Node, error) {
	bs, err := p.Branches()
	if err != nil {
		return mem.Vector_Node{}, err
	}

	return bs.At(i), nil
}

/*
	seq
*/

type chunkedSeq struct{ mem.Any }

func newChunkedSeq(a capnp.Arena, v mem.Vector, i uint32, offset uint8) (chunkedSeq, error) {
	any, err := memutil.Alloc(a)
	if err != nil {
		return chunkedSeq{}, nil
	}

	seq, err := any.NewVectorSeq()
	if err != nil {
		return chunkedSeq{}, err
	}

	if err = seq.SetVector(v); err != nil {
		return chunkedSeq{}, err
	}
	seq.SetIndex(i)
	seq.SetOffset(offset)

	// n, err := vectorArrayFor(v.Any, i)
	// if errors.Is(err, ErrIndexOutOfBounds) {
	// 	_, n, err = newVectorLeafNode(capnp.SingleSegment(nil))
	// }

	return chunkedSeq{
		Any: any,
		// vec:    v,
		// node:   n,
		// i:      i,
		// offset: offset,
	}, err
}

// Value returns the memory value
func (cs chunkedSeq) Value() mem.Any { return cs.Any }

func (cs chunkedSeq) Count() (cnt int, err error) {
	var seq mem.VectorSeq
	if seq, err = cs.Any.VectorSeq(); err != nil {
		return
	}

	var vec mem.Vector
	if vec, err = seq.Vector(); err == nil {
		cnt = int(vec.Count() - (seq.Index() + uint32(seq.Offset())))
	}

	return
}

func (cs chunkedSeq) First() (ww.Any, error) {
	seq, err := cs.Any.VectorSeq()
	if err != nil {
		return nil, err
	}

	node, err := cs.node(seq)
	if err != nil {
		return nil, err
	}

	return AsAny(node.At(int(seq.Offset())))
}

func (cs chunkedSeq) chunkedNext() (Seq, error) {
	seq, err := cs.Any.VectorSeq()
	if err != nil {
		return nil, err
	}

	vec, err := seq.Vector()
	if err != nil {
		return nil, err
	}

	nodelen := uint32(cs.nodeLen(seq, vec))

	// more?
	if i := seq.Index(); i+nodelen < vec.Count() {
		return newChunkedSeq(capnp.SingleSegment(nil), vec, i+nodelen, 0)
	}

	// end of sequence
	return nil, nil
}

func (cs chunkedSeq) Next() (Seq, error) {
	seq, err := cs.Value().VectorSeq()
	if err != nil {
		return nil, err
	}

	vec, err := seq.Vector()
	if err != nil {
		return nil, err
	}

	if int(seq.Offset()+1) < cs.nodeLen(seq, vec) {
		return newChunkedSeq(capnp.SingleSegment(nil), vec, seq.Index(), seq.Offset()+1)
	}

	return cs.chunkedNext()
}

// length of the current array
func (cs chunkedSeq) nodeLen(seq mem.VectorSeq, vec mem.Vector) int {
	cnt := int(vec.Count())
	tailoff := vectorTailoff(cnt)

	// value in tail?
	if int(seq.Index()) >= tailoff {
		return cnt - tailoff
	}

	return width
}

func (cs chunkedSeq) node(seq mem.VectorSeq) (mem.Any_List, error) {
	vec, err := seq.Vector()
	if err != nil {
		return mem.Any_List{}, err
	}

	return apiVectorArrayFor(vec, int(vec.Count()), int(seq.Index()))
}

// prepends each item to the sequence
func (cs chunkedSeq) Conj(items ...ww.Any) (_ Container, err error) {
	var seq Seq = cs
	for _, any := range items {
		if seq, err = Cons(capnp.SingleSegment(nil), any, seq); err != nil {
			break
		}
	}

	return seq, err
}

/*
	vector utils
*/

func newVector(a capnp.Arena, cnt, shift int, root mem.Vector_Node, t mem.Any_List) (DeepPersistentVector, error) {
	val, err := memutil.Alloc(a)
	if err != nil {
		return DeepPersistentVector{}, err
	}

	vec, err := val.NewVector()
	if err != nil {
		return DeepPersistentVector{}, err
	}

	// TODO(performance): lots of calls to capnp.copyStruct and capnp.writePtr, here.
	if err = vec.SetRoot(root); err != nil {
		return DeepPersistentVector{}, err
	}

	if err = vec.SetTail(t); err != nil {
		return DeepPersistentVector{}, err
	}

	vec.SetCount(uint32(cnt))
	vec.SetShift(uint8(shift))

	return DeepPersistentVector{val}, nil
}

func newRootVectorNode(a capnp.Arena) (mem.Vector_Node, error) {
	_, seg, err := capnp.NewMessage(a)
	if err != nil {
		return mem.Vector_Node{}, err
	}

	return mem.NewRootVector_Node(seg)
}

func newVectorNode(a capnp.Arena) (n mem.Vector_Node, bs mem.Vector_Node_List, err error) {
	if n, err = newRootVectorNode(a); err != nil {
		return
	}

	bs, err = n.NewBranches(int32(width))
	return
}

func newVectorNodeWithBranches(a capnp.Arena, bs ...mem.Vector_Node) (n mem.Vector_Node, err error) {
	var branches mem.Vector_Node_List
	if n, branches, err = newVectorNode(a); err != nil {
		return
	}

	for i, b := range bs {
		if err = branches.Set(i, b); err != nil {
			break
		}
	}

	return
}

func newVectorLeafNode(a capnp.Arena) (n mem.Vector_Node, vs mem.Any_List, err error) {
	if n, err = newRootVectorNode(a); err != nil {
		return
	}

	vs, err = n.NewValues(int32(width))
	return
}

func apiVectorArrayFor(vec mem.Vector, cnt, i int) (mem.Any_List, error) {
	// value in tail?
	if i >= vectorTailoff(cnt) {
		return vec.Tail()
	}

	// slow path; value in trie.

	n, err := vec.Root()
	if err != nil {
		return mem.Any_List{}, err
	}

	var bs mem.Vector_Node_List
	for level := vec.Shift(); level > 0; level -= bits {
		if !n.HasBranches() {
			return mem.Any_List{}, Error{
				Cause:   ErrInvalidVectorNode,
				Message: "non-leaf node must branch",
			}
		}

		if bs, err = n.Branches(); err != nil {
			return mem.Any_List{}, err
		}

		n = bs.At((i >> level) & mask)
	}

	if !n.HasValues() {
		return mem.Any_List{}, Error{
			Cause:   ErrInvalidVectorNode,
			Message: "leaf node must contain values",
		}
	}

	return n.Values()
}

// cloneNode deep-copies n.  If lim >= 0, it will only copy the first `lim` elements.
func cloneNode(a capnp.Arena, n mem.Vector_Node, lim int) (mem.Vector_Node, error) {
	switch n.Which() {
	case mem.Vector_Node_Which_branches:
		return cloneBranchNode(a, n, lim)

	case mem.Vector_Node_Which_values:
		return cloneLeafNode(a, n, lim)

	}

	panic(errors.New("cannot clone uninitialized mem.Vector_Node"))
}

func cloneBranchNode(a capnp.Arena, n mem.Vector_Node, lim int) (ret mem.Vector_Node, err error) {
	var bs, rbs mem.Vector_Node_List
	if ret, rbs, err = newVectorNode(a); err != nil {
		return ret, err
	}

	if bs, err = n.Branches(); err != nil {
		return
	}

	if lim < 1 {
		lim = bs.Len()
	}

	for i := 0; i < lim; i++ {
		if err = rbs.Set(i, bs.At(i)); err != nil {
			break
		}
	}

	return
}

func cloneLeafNode(a capnp.Arena, n mem.Vector_Node, lim int) (ret mem.Vector_Node, err error) {
	var vs, rvs mem.Any_List
	if ret, rvs, err = newVectorLeafNode(a); err != nil {
		return
	}

	if vs, err = n.Values(); err != nil {
		return
	}

	for i := 0; i < lim; i++ {
		if err = rvs.Set(i, vs.At(i)); err != nil {
			break
		}
	}

	return
}

func newVectorValueList(a capnp.Arena) (_ mem.Any_List, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	return mem.NewAny_List(seg, width)
}

type item mem.Any

func (i item) Value() mem.Any { return mem.Any(i) }

func tailSlice(cnt int, tail mem.Any_List) []ww.Any {
	items := make([]ww.Any, 0, width)
	for i := 0; i < cnt; i++ {
		items = append(items, item(tail.At(i)))
	}
	return items
}
