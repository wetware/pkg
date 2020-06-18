package server

import (
	goruntime "runtime"
	"sync"
)

// subNode .
type subNode struct {
	Path string
	Node anchorNode
}

// anchorNode in an anchor tree.
type anchorNode struct {
	r *nodeRef
}

// newAnchorTree anchor tree
func newAnchorTree() anchorNode {
	return anchorNode{
		r: newRootNode(),
	}
}

func newRootNode() *nodeRef {
	return newNode(nil, "").ref()
}

// Path from root to the present anchorNode
func (a anchorNode) Path() []string {
	return a.r.Path()
}

// Walk an anchor path
func (a anchorNode) Walk(path []string) anchorNode {
	return anchorNode{
		r: a.r.Walk(path),
	}
}

// Value of the anchor
func (a anchorNode) Value() interface{} {
	a.r.Tx().RLock()
	defer a.r.Tx().RUnlock()

	return a.r.getValUnsafe()
}

// List the anchor's children
func (a anchorNode) List() []subNode {
	// N.B.:  hard-lock because the List() operation may co-occur with a sub-anchor
	//		  creation/deletion.
	a.r.Hard().Lock()
	defer a.r.Hard().Unlock()

	children := make([]subNode, 0, len(a.r.childrenUnsafe()))
	for path, child := range a.r.childrenUnsafe() {
		children = append(children, subNode{Path: path, Node: anchorNode{r: child.ref()}})
	}

	return children
}

// Bind a value to the anchor
func (a anchorNode) Bind(val interface{}) bool {
	a.r.Tx().Lock()
	defer a.r.Tx().Unlock()

	return a.r.setValIfEmpty(val)
}

// nodeRef is a proxy to a node that is responsible for implemented refcounting and gc
// logic.  When anchor is GCed, the underlying node's refcount is decremented.
type nodeRef struct{ n *node }

// transaction lock
func (h nodeRef) Tx() *sync.RWMutex {
	return &h.n.tx
}

// Unsafe - requires locking
func (h nodeRef) setObjUnsafe(val interface{}) {
	h.n.val = val
}

// Unsafe - requires locking
func (h nodeRef) setValIfEmpty(val interface{}) (set bool) {
	if h.n.val == nil {
		set = true
		h.n.val = val
	}

	return
}

// Unsafe - requires locking
func (h nodeRef) getValUnsafe() interface{} {
	return h.n.val
}

// Unsafe - requires locking
func (h nodeRef) childrenUnsafe() map[string]*node {
	return h.n.children
}

// hard lock - prevents updates to children & counter states
func (h nodeRef) Hard() *sync.Mutex {
	return &h.n.µ
}

func (h nodeRef) Path() (parts []string) {
	// zero-allocation filtering of empty path components.
	raw := h.n.path()
	parts = raw[:0]
	for _, segment := range raw {
		if len(segment) > 0 {
			parts = append(parts, segment)
		}
	}
	return
}

func (h nodeRef) Walk(path []string) *nodeRef {
	if len(path) == 0 {
		return h.n.ref()
	}

	h.n.µ.Lock()
	defer h.n.µ.Unlock()

	n, ok := h.n.children[path[0]]
	if !ok {
		n = newNode(h.n, path[0])
		h.n.children[path[0]] = n
		return n.ref().Walk(path[1:]) // Ensure n is garbage-collected.
	}

	// n is already tracked by garbage collector; use concrete `nodeRef`
	return nodeRef{n}.Walk(path[1:])
}

type node struct {
	µ   sync.Mutex
	ctr ctr

	tx  sync.RWMutex
	val interface{}

	name     string
	parent   *node
	children map[string]*node
}

func newNode(parent *node, name string) *node {
	return &node{
		name:     name,
		parent:   parent,
		children: make(map[string]*node),
	}
}

func (n *node) path() []string {
	if n.parent == nil {
		return []string{n.name}
	}

	return append(n.parent.path(), n.name)
}

func (n *node) orphaned() bool {
	n.µ.Lock()
	defer n.µ.Unlock()

	return n.orphanedUnsafe()
}

// Unsafe - requires locking
func (n *node) orphanedUnsafe() bool {
	// - nobody's using it
	// - it has no children
	// - it's not holding an object
	return n.ctr == 0 && len(n.children) == 0 && n.val == nil
}

func (n *node) ref() *nodeRef {
	n.µ.Lock()
	defer n.µ.Unlock()

	n.ctr.Incr()
	h := &nodeRef{n}

	goruntime.SetFinalizer(h, func(h *nodeRef) {
		n.µ.Lock()
		defer n.µ.Unlock()

		n.ctr.Decr()
		if n.orphanedUnsafe() && n.parent != nil {
			go n.parent.rmChild(n.name)
		}
	})

	return h
}

func (n *node) rmChild(childName string) {
	n.µ.Lock()
	defer n.µ.Unlock()

	if child, ok := n.children[childName]; ok && child.orphaned() {
		delete(n.children, childName)
	}
}

type ctr int

func (c *ctr) Incr() { *c++ }
func (c *ctr) Decr() { *c-- }
