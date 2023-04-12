package anchor

import (
	"sync"
	"sync/atomic"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/ww/internal/api/anchor"
)

type node struct {
	sync.Mutex

	refs atomic.Int32

	parent *node
	name   string

	children map[string]*node
	client   *weakClient
	// value api.Value
}

func (n *node) AddRef() *node {
	n.refs.Add(1)
	return n
}

func (n *node) Release() {
	if refs := n.refs.Add(-1); refs == 0 && n.parent != nil {
		defer n.parent.Release()

		n.parent.Lock()
		delete(n.parent.children, n.name)
		n.parent.Unlock()

	} else if refs < 0 {
		panic("no references to release")
	}
}

// Child returns the named child of the current node, creating it if
// it does not exist.
func (n *node) Child(name string) *node {
	n.Lock()
	defer n.Unlock()

	// Fast path;  child exists.
	if child, ok := n.children[name]; ok {
		return child
	}

	// Slow path; create new child.

	if n.children == nil {
		n.children = make(map[string]*node)
	}

	// The child holds the parent reference, releasing it when
	// its own refcount hit zero.
	n.children[name] = &node{
		parent: n.AddRef(),
		name:   name,
	}

	return n.children[name]
}

func (n *node) Anchor() api.Anchor {
	n.Lock()
	defer n.Unlock()

	// Fast path; a server is already running for this node.
	// Node is guaranteed to have r > 1 refs.  This means we
	// can release the refchain after client.AddRef returns.
	if n.client != nil {
		client := n.client.AddRef()
		return api.Anchor(client)
	}

	// Slow path; spin up a new server, assign the weak client,
	// and return the first reference.
	//
	// Node is guaranteed to have r > 0 refs. The refchain must
	// be released after the last client ref has been released.
	// This happens in the Shutdown() method, which is invoked
	// when the last client ref has been released.
	n.AddRef()
	client := capnp.NewClient(&nodeHook{
		Locker:     n,
		ClientHook: api.Anchor_NewServer(server{n}),
	})

	// Set the weak reference; subsequent calls to Anchor() will
	// derive clients from the weakref, incrementing the refcount.
	n.client = (*weakClient)(client.WeakRef())

	// Return first reference to caller;  The RPC connection will
	// take ownership of it and release it when done.  When the
	// client refcount reaches zero, the server will terminate,
	// calling n.Shutdown(), and the weakref will be cleared.
	//
	// Note that this will not necessarily remove n from its
	// parent's children map.  This is handled by the n's rc.Ref
	// field, and only occurs when the node *also* has no children
	// and holds no value.
	return api.Anchor(client)
}

type weakClient capnp.WeakClient

func (wc *weakClient) AddRef() capnp.Client {
	c, ok := (*capnp.WeakClient)(wc).AddRef()

	// Shutdown() ensures this never happens.
	if !ok || c == (capnp.Client{}) {
		panic("nil or released WeakClient")
	}

	return c
}

type nodeHook struct {
	sync.Locker
	capnp.ClientHook
}

func (h *nodeHook) Shutdown() {
	h.Lock()
	defer h.Unlock()

	h.ClientHook.Shutdown()
}
