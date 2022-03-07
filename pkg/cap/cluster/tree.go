package cluster

import (
	"sync"
	"sync/atomic"

	syncutil "github.com/lthibault/util/sync"
	api "github.com/wetware/ww/internal/api/cluster"
)

type ServerAnchor interface {
	Anchor() api.Anchor
}

type releaseFunc func()

type node struct {
	ref ref

	Name   string
	parent *node

	mu       sync.RWMutex
	children map[string]*node

	Server ServerAnchor

	Value atomic.Value
}

func (n *node) Path() []string {
	if n.parent == nil {
		return make([]string, 0, 16) // best-effort pre-alloc
	}

	return append(n.parent.Path(), n.Name)
}

func (n *node) Acquire() *node {
	n.mu.RLock()
	defer n.mu.RUnlock()

	n.ref.Incr()
	return n
}

func (n *node) Release() {
	if n.parent != nil {
		defer n.parent.Release()
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.ref.Decr()

	for k, v := range n.children {
		if v.ref.Zero() {
			v.mu.Lock()

			// still zero?
			if v.ref.Zero() {
				delete(n.children, k)
			}

			v.mu.Unlock()
		}
	}
}

func (n *node) Children() (map[string]*node, releaseFunc) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if len(n.children) == 0 {
		return nil, func() {}
	}

	m := make(map[string]*node, len(n.children)) // TODO:  pool
	for k, v := range n.children {
		n.ref.Incr()
		m[k] = v.Acquire()
	}

	return m, func() {
		for _, v := range m { // TODO:  return n.cs to pool?
			v.Release()
		}
	}
}

func (n *node) Walk(path []string, serverCreator func(*node) ServerAnchor) (u *node) {
	if len(path) == 0 {
		return n
	}

	n.mu.Lock()
	if n.children == nil {
		n.children = make(map[string]*node, 1)
	}

	if u = n.children[path[0]]; u == nil {
		u = &node{Name: path[0], parent: n}
		if serverCreator != nil {
			u.Server = serverCreator(u)
		}
		n.children[path[0]] = u
	}
	u.Acquire()
	n.mu.Unlock()

	return u.Walk(path[1:], serverCreator)
}

type ref syncutil.Ctr

func (r *ref) Incr()      { (*syncutil.Ctr)(r).Incr() }
func (r *ref) Decr()      { (*syncutil.Ctr)(r).Decr() }
func (r *ref) Zero() bool { return (*syncutil.Ctr)(r).Int() == 0 }
