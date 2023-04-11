package anchor

import (
	"sync"
	"sync/atomic"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/ww/internal/api/anchor"
)

type node struct {
	*nodestate
	release capnp.ReleaseFunc
}

func mknode(release capnp.ReleaseFunc) *node {
	state := &nodestate{}
	state.refs.Add(1)

	return &node{
		nodestate: state,
		release:   release,
	}
}

func (n *node) AddRef() *node {
	if n.release == nil {
		panic("AddRef() to node after garbage-collection")
	}

	n.refs.Add(1)
	return &node{
		nodestate: n.nodestate,
		release:   n.release,
	}
}

func (n *node) Release() {
	if n.refs.Add(^uint32(0)) == 0 {
		n.release()
		*n = node{}
	}
}

// Child returns the named child of the current node, creating it if
// it does not exist.
func (n *node) Child(name string) *node {
	n.Lock()
	defer n.Unlock()

	// Fast path;  child exists.
	if child, ok := n.children[name]; ok {
		// A parent MUST NOT be released until its children are
		// all released; increment the child's refcount.
		return child.AddRef()
	}

	// Slow path; create new child.

	if n.children == nil {
		n.children = make(map[string]*node)
	}

	// The child holds the parent reference, releasing it when
	// its own refcount hit zero.
	parent := n.AddRef()
	n.children[name] = mknode(func() {
		defer parent.Release()

		n.Lock()
		delete(n.children, name)
		n.Unlock()
	})

	return n.children[name]
}

// Anchor produces an anchor capability for the node, stealing
// n's reference and releasing it when the Anchor's underlying
// client is released.  For this reason, callers MUST NOT call
// anchor on an existing node without first creating a new ref.
func (n *node) Anchor() api.Anchor {
	n.Lock()
	defer n.Unlock()

	// Fast path; a server is already running for this node.
	// Node is guaranteed to have r > 1 refs.  This means we
	// can release the refchain after client.AddRef returns.
	if n.client.Exists() {
		defer n.Release()

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
	client := capnp.NewClient(&nodeHook{
		nodestate:  n.nodestate,
		ClientHook: api.Anchor_NewServer(server{n}),
	})

	// Set the weak reference; subsequent calls to Anchor() will
	// derive clients from the weakref, incrementing the refcount.
	n.client = weakClient{
		WeakClient: client.WeakRef(),
		release:    n.Release,
	}

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

type nodestate struct {
	sync.Mutex
	refs     atomic.Uint32
	children map[string]*node
	client   weakClient
	// value api.Value
}

type weakClient struct {
	*capnp.WeakClient
	release capnp.ReleaseFunc
}

func (wc weakClient) AddRef() capnp.Client {
	c, ok := wc.WeakClient.AddRef()

	// Shutdown() ensures this never happens.
	if !ok || c == (capnp.Client{}) {
		panic("nil or released WeakClient")
	}

	return c
}

func (wc *weakClient) Release() {
	wc.release()
	*wc = weakClient{}
}

func (wc weakClient) Exists() bool {
	return wc.WeakClient != nil
}

type nodeHook struct {
	*nodestate
	capnp.ClientHook
}

func (h *nodeHook) Shutdown() {
	h.Lock()
	defer h.Unlock()

	h.ClientHook.Shutdown()
}
