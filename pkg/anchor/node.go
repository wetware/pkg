package anchor

import (
	"sync"
	"sync/atomic"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/ww/internal/api/anchor"
)

type Node struct {
	sync.Mutex

	refs atomic.Int32

	parent *Node
	name   string

	children map[string]*Node
	client   *weakClient
	// value api.Value
}

func (n *Node) AddRef() *Node {
	n.refs.Add(1)
	return n
}

func (n *Node) Release() {
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
func (n *Node) Child(name string) *Node {
	n.Lock()
	defer n.Unlock()

	// Fast path;  child exists.
	if child, ok := n.children[name]; ok {
		return child
	}

	// Slow path; create new child.

	if n.children == nil {
		n.children = make(map[string]*Node)
	}

	// The child holds the parent reference, releasing it when
	// its own refcount hit zero.
	n.children[name] = &Node{
		parent: n.AddRef(),
		name:   name,
	}

	return n.children[name]
}

func (n *Node) Anchor() Anchor {
	n.Lock()
	defer n.Unlock()

	// Fast path; a server is already running for this node.
	// Node is guaranteed to have r > 1 refs.  This means we
	// can release the refchain after client.AddRef returns.
	if n.client != nil {
		client := n.client.AddRef()
		return Anchor(client)
	}

	// Slow path; spin up a new server, assign the weak client,
	// and return the first reference.
	//
	// Node is guaranteed to have r > 0 refs. The refchain must
	// be released after the last client ref has been released.
	// This happens in the Shutdown() method, which is invoked
	// when the last client ref has been released.

	server := server{n.AddRef()}
	client := capnp.NewClient(&nodeHook{
		Locker:     n,
		ClientHook: api.Anchor_NewServer(server),
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
	return Anchor(client)
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
