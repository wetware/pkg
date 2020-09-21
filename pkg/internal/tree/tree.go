// Package tree contains the anchor tree
package tree

import (
	goruntime "runtime"
	"sync"

	"github.com/wetware/ww/pkg/mem"
)

// Node in an anchor tree.
type Node struct{ *nodeRef }

// New anchor tree
func New() Node {
	return Node{newRootNode()}
}

func newRootNode() *nodeRef {
	return newNode(nil, "").ref()
}

// Path from root to the present Node
func (a Node) Path() []string {
	return a.nodeRef.Path()
}

// Walk an anchor path
func (a Node) Walk(path []string) Node {
	return Node{a.nodeRef.Walk(path)}
}

// List the anchor's children
func (a Node) List() []Node {
	// N.B.:  hard-lock because the List() operation may co-occur with a sub-anchor
	//		  creation/deletion.
	a.nodeRef.Hard().Lock()
	defer a.nodeRef.Hard().Unlock()

	children := make([]Node, 0, len(a.nodeRef.children))
	for _, child := range a.nodeRef.children {
		children = append(children, Node{child.ref()})
	}

	return children
}

// Load API value
func (a Node) Load() mem.Value {
	a.Tx().RLock()
	defer a.Tx().RUnlock()

	return a.val
}

// Store API value
func (a Node) Store(val mem.Value) bool {
	a.Tx().Lock()
	defer a.Tx().Unlock()

	if val.Nil() || a.val.Nil() {
		a.val = val
		return true
	}

	return false
}

// nodeRef is a proxy to a node that is responsible for implemented refcounting and gc
// logic.  When anchor is GCed, the underlying node's refcount is decremented.
type nodeRef struct{ *node }

// transaction lock
func (h nodeRef) Tx() *sync.RWMutex {
	return &h.tx
}

// hard lock - prevents updates to children & counter states
func (h nodeRef) Hard() *sync.Mutex {
	return &h.mu
}

func (h nodeRef) Path() (parts []string) {
	// zero-allocation filtering of empty path components.
	raw := h.path()
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
		return h.ref()
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	n, ok := h.children[path[0]]
	if !ok {
		n = newNode(h.node, path[0])
		h.children[path[0]] = n
		return n.ref().Walk(path[1:]) // Ensure n is garbage-collected.
	}

	// n is already tracked by garbage collector; use concrete `nodeRef`
	return nodeRef{n}.Walk(path[1:])
}

type node struct {
	mu  sync.Mutex
	ctr int

	tx  sync.RWMutex
	val mem.Value

	Name     string
	parent   *node
	children map[string]*node
}

func newNode(parent *node, name string) *node {
	return &node{
		Name:     name,
		parent:   parent,
		children: make(map[string]*node),
	}
}

func (n *node) path() []string {
	if n.parent == nil {
		return []string{n.Name}
	}

	return append(n.parent.path(), n.Name)
}

func (n *node) orphaned() bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	return n.orphanedUnsafe()
}

// Unsafe - requires locking
func (n *node) orphanedUnsafe() bool {
	// - nobody's using it
	// - it has no children
	// - it's not holding an object
	return n.ctr == 0 && len(n.children) == 0 && n.val.Nil()
}

func (n *node) ref() *nodeRef {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.ctr++
	ref := &nodeRef{n}
	goruntime.SetFinalizer(ref, gc)

	return ref
}

func gc(n *nodeRef) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.ctr--
	if n.orphanedUnsafe() && n.parent != nil {
		if child, ok := n.children[n.Name]; ok && child.orphanedUnsafe() {
			delete(n.children, n.Name)
		}
	}
}
