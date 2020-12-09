package core

import (
	"errors"
	"fmt"
	"strings"

	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/mem"
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
	EmptyVector PersistentVector

	_ Vector = (*PersistentVector)(nil)
)

func init() {
	root, _, err := newVectorNode(capnp.SingleSegment([]byte{}))
	if err != nil {
		panic(err)
	}

	tail, err := newVectorValueList(capnp.SingleSegment([]byte{}), 0)
	if err != nil {
		panic(err)
	}

	_, vec, err := newVector(capnp.SingleSegment(nil), 0, bits, root, tail)
	if err != nil {
		panic(err)
	}
	EmptyVector.Raw = vec.Raw
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
}

// PersistentVector is a persistent, immutable vector.
type PersistentVector struct{ mem.Value }

// NewVector creates a vector containing the supplied values.
func NewVector(a capnp.Arena, vs ...ww.Any) (vec PersistentVector, err error) {
	if vec = EmptyVector; len(vs) > 0 {
		vec, err = vec.conj(vs...)
	}

	return
}

// Invoke is equivalent to `EntryAt`.
func (v PersistentVector) Invoke(args ...ww.Any) (ww.Any, error) {
	if nargs := len(args); nargs != 1 {
		return nil, fmt.Errorf("%w: got %d, want at-least 1", ErrArity, nargs)
	}

	switch idx := args[0]; idx.MemVal().Type() {
	case api.Value_Which_i64:
		return v.EntryAt(int(idx.MemVal().Raw.I64()))
	case api.Value_Which_bigInt:
		// TODO(performance):  can we use unsafe.Pointer here?
		if bi := idx.(BigInt).BigInt(); bi.IsInt64() {
			return v.EntryAt(int(bi.Int64()))
		}

		fallthrough

	default:
		return nil, fmt.Errorf("%s is not an integer type", idx.MemVal().Type())
	}
}

// Render the vector in a human-readable format.
func (v PersistentVector) Render() (string, error) {
	return v.render(func(any ww.Any) (string, error) {
		return Render(any)
	})
}

func (v PersistentVector) render(f func(ww.Any) (string, error)) (string, error) {
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

		s, err := f(val)
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
func (v PersistentVector) Count() (cnt int, err error) {
	_, cnt, err = v.count()
	return
}

func (v PersistentVector) count() (vec api.Vector, cnt int, err error) {
	if vec, err = v.Raw.Vector(); err != nil {
		return
	}

	cnt = int(vec.Count())
	return
}

// Conj returns a new vector with items appended.
func (v PersistentVector) Conj(items ...ww.Any) (Container, error) { return v.conj(items...) }

func (v PersistentVector) conj(items ...ww.Any) (PersistentVector, error) {
	for _, item := range items {
		vec, cnt, err := v.count()
		if err != nil {
			return PersistentVector{}, err
		}

		if v, err = v.cons(vec, cnt, item); err != nil {
			return PersistentVector{}, err
		}
	}

	return v, nil
}

// EntryAt returns the item at given index. Returns error if the index
// is out of range.
func (v PersistentVector) EntryAt(i int) (ww.Any, error) {
	vs, err := v.arrayFor(i)
	if err != nil {
		return nil, err
	}

	return AsAny(mem.Value{Raw: vs.At(i & mask)})
}

// Assoc returns a new vector with the value at given index updated.
// Returns error if the index is out of range.
func (v PersistentVector) Assoc(i int, val ww.Any) (Vector, error) {
	// https://github.com/clojure/clojure/blob/0b73494c3c855e54b1da591eeb687f24f608f346/src/jvm/clojure/lang/PersistentVector.java#L121

	vec, cnt, err := v.count()
	if err != nil {
		return nil, err
	}

	// update?
	if i >= 0 && i < cnt {
		return v.update(vec, cnt, i, val)
	}

	// append?
	if i == cnt {
		return v.cons(vec, cnt, val)
	}

	return nil, ErrIndexOutOfBounds
}

// Pop returns a new vector without the last item in v
func (v PersistentVector) Pop() (Vector, error) {
	if _, vec, err := v.pop(); err != ErrIllegalState {
		return vec, err
	}

	return nil, fmt.Errorf("%w: cannot pop from empty vector", ErrIllegalState)
}

// Seq presents the vector as an iterable sequence.
func (v PersistentVector) Seq() (Seq, error) {
	vec, err := v.Raw.Vector()
	if err != nil {
		return nil, err
	}

	if vec.Count() == 0 {
		return EmptyList, nil
	}

	return newChunkedSeq(nil, vec, 0, 0)
}

func popVectorTail(level, cnt int, n api.Vector_Node) (ret api.Vector_Node, ok bool, err error) {
	subidx := ((cnt - 2) >> level) & mask
	if level > 5 {
		var bs api.Vector_Node_List
		if bs, err = n.Branches(); err != nil {
			return
		}

		var newchild api.Vector_Node
		switch newchild, ok, err = popVectorTail(level-5, cnt, bs.At(subidx)); {
		case err != nil, !ok && subidx == 0:
			return
		}

		if ret, err = cloneBranchNode(capnp.SingleSegment(nil), n, -1); err != nil {
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

func (v PersistentVector) arrayFor(i int) (api.Value_List, error) {
	// See:  https://github.com/clojure/clojure/blob/0b73494c3c855e54b1da591eeb687f24f608f346/src/jvm/clojure/lang/PersistentVector.java#L97-L113
	vec, cnt, err := v.count()
	if err == nil {
		if i < 0 || i >= cnt {
			return api.Value_List{}, ErrIndexOutOfBounds
		}
	}

	return apiVectorArrayFor(vec, int(cnt), i)
}

func apiVectorArrayFor(vec api.Vector, cnt, i int) (api.Value_List, error) {
	// value in tail?
	if i >= vectorTailoff(cnt) {
		return vec.Tail()
	}

	// slow path; value in trie.

	n, err := vec.Root()
	if err != nil {
		return api.Value_List{}, err
	}

	var bs api.Vector_Node_List
	for level := vec.Shift(); level > 0; level -= bits {
		if !n.HasBranches() {
			return api.Value_List{}, Error{
				Cause:   ErrInvalidVectorNode,
				Message: "non-leaf node must branch",
			}
		}

		if bs, err = n.Branches(); err != nil {
			return api.Value_List{}, err
		}

		n = bs.At((i >> level) & mask)
	}

	if !n.HasValues() {
		return api.Value_List{}, Error{
			Cause:   ErrInvalidVectorNode,
			Message: "leaf node must contain values",
		}
	}

	return n.Values()
}

func (PersistentVector) update(vec api.Vector, cnt, i int, any ww.Any) (Vector, error) {
	root, err := vec.Root()
	if err != nil {
		return nil, err
	}

	tail, err := vec.Tail()
	if err != nil {
		return nil, err
	}

	// room in tail?
	if i >= vectorTailoff(cnt) {
		// Object[] newTail = new Object[tail.length];
		// System.arraycopy(tail, 0, newTail, 0, tail.length);
		if tail, err = cloneValueList(capnp.SingleSegment(nil), tail); err != nil {
			return nil, err
		}

		// newTail[i & 0x01f] = any;
		if err = tail.Set(i&mask, any.MemVal().Raw); err != nil {
			return nil, err
		}
	} else {
		if root, err = apiVectorAssoc(int(vec.Shift()), root, i, any); err != nil {
			return nil, err
		}
	}

	_, res, err := newVector(capnp.SingleSegment(nil),
		cnt,
		int(vec.Shift()),
		root,
		tail,
	)
	return res, err
}

func (v PersistentVector) cons(vec api.Vector, cnt int, any ww.Any) (_ PersistentVector, err error) {
	shift := int(vec.Shift())

	var root api.Vector_Node
	if root, err = vec.Root(); err != nil {
		return
	}

	var tail api.Value_List
	if tail, err = vec.Tail(); err != nil {
		return
	}

	/*
		Fast path; room in tail?
	*/
	if cnt-vectorTailoff(cnt) < width {
		var newtail api.Value_List
		if newtail, err = newVectorValueList(capnp.SingleSegment(nil), tail.Len()+1); err != nil {
			return
		}

		if err = copyVectorTail(newtail, tail, cnt&mask); err != nil {
			return
		}

		if err = newtail.Set(cnt&mask, any.MemVal().Raw); err != nil {
			return
		}

		_, res, err := newVector(capnp.SingleSegment(nil),
			cnt+1,
			shift,
			root,
			newtail)
		return res, err
	}

	/*
		Slow path; push to tree
	*/

	var newroot api.Vector_Node

	// Wrap the tail in a node so that we can push it into the trie.
	var tailnode api.Vector_Node
	if tailnode, err = v.newLeafNode(capnp.SingleSegment(nil), tail); err != nil {
		return
	}

	// Overflow root?
	if (cnt >> bits) > (1 << shift) {
		if newroot, err = newRootVectorNode(capnp.SingleSegment(nil)); err != nil {
			return
		}

		var array api.Vector_Node_List
		if array, err = newroot.NewBranches(2); err != nil {
			return
		}

		// first branch points to old root
		if err = array.Set(0, root); err != nil {
			return
		}

		// second branch points to former tail
		var path api.Vector_Node
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

	newtail, err := v.newTail(capnp.SingleSegment(nil), any)
	if err != nil {
		return
	}

	_, res, err := newVector(capnp.SingleSegment(nil),
		cnt+1,
		shift,
		newroot,
		newtail)
	return res, err
}

func (PersistentVector) newTail(a capnp.Arena, item ww.Any) (t api.Value_List, err error) {
	if t, err = newVectorValueList(capnp.SingleSegment(nil), 1); err == nil {
		err = t.Set(0, item.MemVal().Raw)
	}

	return t, err
}

func apiVectorAssoc(level int, n api.Vector_Node, i int, v ww.Any) (ret api.Vector_Node, err error) {
	if ret, err = cloneNode(capnp.SingleSegment(nil), n, -1); err != nil {
		return
	}

	// is leaf?
	if level == 0 {
		err = setNodeValue(ret, i&mask, v)
		return
	}

	// else assoc branch

	var bs api.Vector_Node_List
	if bs, err = n.Branches(); err != nil {
		return
	}

	subidx := (i >> level) & mask
	if n, err = apiVectorAssoc(level-bits, bs.At(subidx), i, v); err != nil {
		return
	}

	err = setNodeBranch(ret, n, subidx)
	return

}

func vectorTailoff(cnt int) int {
	if cnt < width {
		return 0
	}

	return ((cnt - 1) >> bits) << bits
}

func (v PersistentVector) pop() (vec api.Vector, _ Vector, err error) {
	var cnt int
	if vec, cnt, err = v.count(); err != nil {
		return
	}

	switch cnt {
	case 0:
		err = ErrIllegalState
		return
	case 1:
		return vec, EmptyVector, nil
	}

	var root api.Vector_Node
	if root, err = vec.Root(); err != nil {
		return
	}

	// more than one item in the tail?
	var newtail api.Value_List
	if pos := cnt - vectorTailoff(cnt); pos > 1 {
		var tail api.Value_List
		if tail, err = vec.Tail(); err != nil {
			return
		}

		if newtail, err = newVectorValueList(capnp.SingleSegment(nil), pos-1); err != nil {
			return
		}

		if err = copyVectorTail(newtail, tail, pos-1); err != nil {
			return
		}

		return newVector(capnp.SingleSegment(nil),
			cnt-1,
			int(vec.Shift()),
			root,
			newtail)
	}

	/*
		slow path; single item in tail => fetch tail node from trie
	*/

	if newtail, err = v.arrayFor(cnt - 2); err != nil {
		return
	}

	shift := int(vec.Shift())
	var ok bool
	var newroot api.Vector_Node
	if newroot, ok, err = popVectorTail(shift, cnt, root); err != nil {
		return
	}

	// null node?
	if !ok {
		// 	newroot = EMPTY_NODE;
		if newroot, err = newVectorNodeWithBranches(capnp.SingleSegment(nil)); err != nil {
			return
		}
	}

	var bs api.Vector_Node_List
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

func (v PersistentVector) newPath(level int, node api.Vector_Node) (ret api.Vector_Node, err error) {
	if level == 0 {
		return node, nil
	}

	if ret, err = newRootVectorNode(capnp.SingleSegment(nil)); err != nil {
		return
	}

	var array api.Vector_Node_List
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

	var path api.Vector_Node
	if path, err = v.newPath(level-bits, node); err == nil {
		err = array.Set(0, path)
	}

	return
}

func (v PersistentVector) pushTail(level, cnt int, parent, tailnode api.Vector_Node) (_ api.Vector_Node, err error) {
	// if parent is leaf => insert node,
	//   else does it map to an existing child? => nodeToInsert = pushNode one more level
	//   else => alloc new path
	//
	// return nodeToInsert placed in parent

	var nodeToInsert api.Vector_Node
	subidx := ((cnt - 1) >> level) & mask

	// parent is leaf?
	if level == bits {
		nodeToInsert = tailnode
	} else {
		var child api.Vector_Node
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

	return parent, setNodeBranch(parent, nodeToInsert, subidx)
}

func nullNode(n api.Vector_Node) bool {
	return !n.HasBranches() && !n.HasValues()
}

func nodeNotNull(n api.Vector_Node) bool {
	return n.HasBranches() || n.HasValues()
}

func getChild(p api.Vector_Node, i int) (api.Vector_Node, error) {
	bs, err := p.Branches()
	if err != nil {
		return api.Vector_Node{}, err
	}

	return bs.At(i), nil
}

/*
	seq
*/

type chunkedSeq struct {
	mem.Value
	newArena func() capnp.Arena

	// vec       Vector
	// node      api.Value_List
	// i, offset int
}

func newChunkedSeq(newArena func() capnp.Arena, v api.Vector, i, offset uint32) (chunkedSeq, error) {
	if newArena == nil {
		newArena = func() capnp.Arena { return capnp.SingleSegment(nil) }
	}

	val, err := mem.NewValue(newArena())
	if err != nil {
		return chunkedSeq{}, nil
	}

	seq, err := val.Raw.NewVectorSeq()
	if err != nil {
		return chunkedSeq{}, err
	}

	if err = seq.SetVector(v); err != nil {
		return chunkedSeq{}, err
	}
	seq.SetIndex(i)
	seq.SetOffset(offset)

	// n, err := vectorArrayFor(v.MemVal(), i)
	// if errors.Is(err, ErrIndexOutOfBounds) {
	// 	_, n, err = newVectorLeafNode(capnp.SingleSegment(nil))
	// }

	return chunkedSeq{
		Value:    val,
		newArena: newArena,
		// vec:    v,
		// node:   n,
		// i:      i,
		// offset: offset,
	}, err
}

func (cs chunkedSeq) Count() (cnt int, err error) {
	var seq api.VectorSeq
	if seq, err = cs.Raw.VectorSeq(); err != nil {
		return
	}

	var vec api.Vector
	if vec, err = seq.Vector(); err == nil {
		cnt = int(vec.Count() - (seq.Index() - seq.Offset()))
	}

	return
}

func (cs chunkedSeq) First() (ww.Any, error) {
	seq, err := cs.Raw.VectorSeq()
	if err != nil {
		return nil, err
	}

	node, err := cs.node(seq)
	if err != nil {
		return nil, err
	}

	return AsAny(mem.Value{Raw: node.At(cs.offset(seq))})
}

func (cs chunkedSeq) Next() (Seq, error) {
	seq, err := cs.Raw.VectorSeq()
	if err != nil {
		return nil, err
	}

	node, err := cs.node(seq)
	if err != nil {
		return nil, err
	}

	if cs.offset(seq)+1 < node.Len() {
		val, err := mem.NewValue(cs.newArena())
		if err != nil {
			return chunkedSeq{}, nil
		}

		if err = val.Raw.SetVectorSeq(seq); err != nil {
			return chunkedSeq{}, nil
		}

		if seq, err = val.Raw.VectorSeq(); err == nil {
			seq.SetOffset(seq.Offset() + 1)
		}

		return chunkedSeq{
			Value:    val,
			newArena: cs.newArena,
		}, nil
	}

	return cs.chunkedNext()
}

func (cs chunkedSeq) index(seq api.VectorSeq) int  { return int(seq.Index()) }
func (cs chunkedSeq) offset(seq api.VectorSeq) int { return int(seq.Offset()) }

func (cs chunkedSeq) node(seq api.VectorSeq) (api.Value_List, error) {
	vec, err := seq.Vector()
	if err != nil {
		return api.Value_List{}, err
	}

	return apiVectorArrayFor(vec, int(vec.Count()), cs.index(seq))
}

func (cs chunkedSeq) chunkedNext() (Seq, error) {
	seq, err := cs.Raw.VectorSeq()
	if err != nil {
		return nil, err
	}

	vec, err := seq.Vector()
	if err != nil {
		return nil, err
	}

	node, err := cs.node(seq)
	if err != nil {
		return nil, err
	}

	// more?
	if i := seq.Index() + uint32(node.Len()); i < vec.Count() {
		return newChunkedSeq(cs.newArena, vec, i, 0)
	}

	// end of sequence
	return nil, nil
}

// prepends each item to the sequence
func (cs chunkedSeq) Conj(items ...ww.Any) (_ Container, err error) {
	var seq Seq = cs
	for _, any := range items {
		if seq, err = Cons(cs.newArena(), any, seq); err != nil {
			break
		}
	}

	return seq, err
}

/*
	vector utils
*/

func newVector(a capnp.Arena, cnt, shift int, root api.Vector_Node, t api.Value_List) (api.Vector, PersistentVector, error) {
	val, err := mem.NewValue(a)
	if err != nil {
		return api.Vector{}, PersistentVector{}, err
	}

	vec, err := val.Raw.NewVector()
	if err != nil {
		return api.Vector{}, PersistentVector{}, err
	}

	if err = vec.SetRoot(root); err != nil {
		return api.Vector{}, PersistentVector{}, err
	}

	if err = vec.SetTail(t); err != nil {
		return api.Vector{}, PersistentVector{}, err
	}

	vec.SetCount(uint32(cnt))
	vec.SetShift(uint8(shift))

	return vec, PersistentVector{val}, nil
}

func newRootVectorNode(a capnp.Arena) (api.Vector_Node, error) {
	_, seg, err := capnp.NewMessage(a)
	if err != nil {
		return api.Vector_Node{}, err
	}

	return api.NewRootVector_Node(seg)
}

func newVectorNode(a capnp.Arena) (n api.Vector_Node, bs api.Vector_Node_List, err error) {
	if n, err = newRootVectorNode(a); err != nil {
		return
	}

	bs, err = n.NewBranches(int32(width))
	return
}

func newVectorNodeWithBranches(a capnp.Arena, bs ...api.Vector_Node) (n api.Vector_Node, err error) {
	var branches api.Vector_Node_List
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

func newVectorLeafNode(a capnp.Arena) (n api.Vector_Node, vs api.Value_List, err error) {
	if n, err = newRootVectorNode(a); err != nil {
		return
	}

	vs, err = n.NewValues(int32(width))
	return
}

// vs is always the old tail, which is now being pushed into the trie.
func (PersistentVector) newLeafNode(a capnp.Arena, vs api.Value_List) (n api.Vector_Node, err error) {
	if n, err = newRootVectorNode(a); err == nil {
		err = n.SetValues(vs)
	}

	return
}

func setNodeBranch(p, n api.Vector_Node, i int) error {
	bs, err := p.Branches()
	if err != nil {
		return err
	}

	return bs.Set(i, n)
}

func setNodeValue(n api.Vector_Node, i int, any ww.Any) error {
	vs, err := n.Values()
	if err != nil {
		return err
	}

	return vs.Set(i, any.MemVal().Raw)
}

// cloneNode deep-copies n.  If lim >= 0, it will only copy the first `lim` elements.
func cloneNode(a capnp.Arena, n api.Vector_Node, lim int) (api.Vector_Node, error) {
	if n.HasBranches() {
		return cloneBranchNode(a, n, lim)
	}

	if n.HasValues() {
		return cloneLeafNode(a, n, lim)
	}

	panic(errors.New("cannot clone uninitialized api.Vector_Node"))
}

func cloneBranchNode(a capnp.Arena, n api.Vector_Node, lim int) (ret api.Vector_Node, err error) {
	var bs, rbs api.Vector_Node_List
	if ret, rbs, err = newVectorNode(a); err != nil {
		return ret, err
	}

	if bs, err = n.Branches(); err != nil {
		return
	}

	if lim < 0 {
		lim = bs.Len()
	}

	for i := 0; i < lim; i++ {
		if err = rbs.Set(i, bs.At(i)); err != nil {
			break
		}
	}

	return
}

func cloneLeafNode(a capnp.Arena, n api.Vector_Node, lim int) (ret api.Vector_Node, err error) {
	var vs, rvs api.Value_List
	if ret, rvs, err = newVectorLeafNode(a); err != nil {
		return
	}

	if vs, err = n.Values(); err != nil {
		return
	}

	if lim < 0 {
		lim = vs.Len()
	}

	for i := 0; i < lim; i++ {
		if err = rvs.Set(i, vs.At(i)); err != nil {
			break
		}
	}

	return
}

func cloneValueList(a capnp.Arena, vs api.Value_List) (ret api.Value_List, err error) {
	if ret, err = newVectorValueList(a, vs.Len()); err != nil {
		return
	}

	for i := 0; i < vs.Len(); i++ {
		if err = ret.Set(i, vs.At(i)); err != nil {
			break
		}
	}

	return
}

func newVectorValueList(a capnp.Arena, size int) (_ api.Value_List, err error) {
	var seg *capnp.Segment
	if _, seg, err = capnp.NewMessage(a); err != nil {
		return
	}

	return api.NewValue_List(seg, int32(size))
}

func copyVectorTail(dst, src api.Value_List, lim int) (err error) {
	for i := 0; i < lim; i++ {
		if err = dst.Set(i, src.At(i)); err != nil {
			break
		}
	}

	return
}
