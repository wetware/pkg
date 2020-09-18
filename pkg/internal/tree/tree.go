// Package tree contains the anchor tree
package tree

import (
	goruntime "runtime"
	"sync"

	"github.com/wetware/ww/internal/api"
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

	children := make([]Node, 0, len(a.nodeRef.childrenUnsafe()))
	for _, child := range a.nodeRef.childrenUnsafe() {
		children = append(children, Node{child.ref()})
	}

	return children
}

// Load API value
func (a Node) Load() *api.Value {
	a.Tx().RLock()
	defer a.Tx().RUnlock()

	return a.getValUnsafe()
}

// Store API value
func (a Node) Store(val *api.Value) bool {
	a.Tx().Lock()
	defer a.Tx().Unlock()

	return a.setValIfEmpty(val)
}

// nodeRef is a proxy to a node that is responsible for implemented refcounting and gc
// logic.  When anchor is GCed, the underlying node's refcount is decremented.
type nodeRef struct{ *node }

// transaction lock
func (h nodeRef) Tx() *sync.RWMutex {
	return &h.tx
}

// Unsafe - requires locking
func (h nodeRef) setObjUnsafe(val *api.Value) {
	h.val = val
}

// Unsafe - requires locking
func (h nodeRef) setValIfEmpty(val *api.Value) bool {
	if val == nil || h.val == nil {
		h.val = val
		return true
	}

	return false
}

// Unsafe - requires locking
func (h nodeRef) getValUnsafe() *api.Value {
	return h.val
}

// Unsafe - requires locking
func (h nodeRef) childrenUnsafe() map[string]*node {
	return h.children
}

// hard lock - prevents updates to children & counter states
func (h nodeRef) Hard() *sync.Mutex {
	return &h.µ
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

	h.µ.Lock()
	defer h.µ.Unlock()

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
	µ   sync.Mutex
	ctr ctr

	tx  sync.RWMutex
	val *api.Value

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
			go n.parent.rmChild(n.Name)
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
