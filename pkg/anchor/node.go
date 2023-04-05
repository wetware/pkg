package anchor

import (
	"context"
	"sync"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/ww/internal/api/anchor"
	"zenhack.net/go/util/rc"
)

// node is an element in the anchor tree. Anchor nodes are
// created dynamically upon traversal, such that all calls
// walk must succeed. Nodes are likewise deallocated based
// on the following rules:
//
//  1. The node has no children
//  2. The node has no client references
//  3. The node has no value
//
// To enforce these invariants, a hierarchical refcounting
// scheme is employed:
//
//  1. A node is created with an initial refcount of 1.
//  2. Each time a node is traversed, add 1 to its refcount.
//  3. When releasing a ref on node n, also release a ref on
//     n's parent.
//  4. When creating an Anchor capability or a value, endow
//     it with the ref produced by the preceeding traversal,
//     and ensure that this ref is released when the object
//     reaches the end of its lifetime.
//  5. After acquiring a ref to an existing Anchor or value,
//     release the ref produced by the preceeding traversal.
//
// These five rules ensure that the lifetime invariants and
// ordering invariants are maintained.
type node struct {
	rc *rc.Ref[nodestate]
}

func (n node) state() *nodestate {
	return n.rc.Value()
}

func (n node) Shutdown() {
	// anchorRef holds the lock when shutting down.  Note that
	// the Release() method calls an rc.Ref.Release(), meaning
	// that the parent is released iff the client had the last
	// remaining reference.
	n.state().client.Release() // rule 4 (Anchor)
	n.rc.Release()
}

func (n node) Ls(ctx context.Context, call api.Anchor_ls) error {
	panic("NOT IMPLEMENTED")
}

// FIXME:  there is currently a vector for resource-exhaustion attacks.
// We don't enforce a maximum depth on anchors, nor do we enforce a max
// number of children per node. An attacker can exploit this by walking
// an arbitrarily long path and/or by creating arbitrarily many anchors,
// ultimately exhausting the attacker's memory.
func (n node) Walk(ctx context.Context, call api.Anchor_walk) error {
	path := newPath(call)
	if path.Err() != nil {
		return path.Err()
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	// Do we skip the iteration loop?  If so, we need to make sure that
	// we increment the refcount on n, since it will be released by the
	// call to n.Anchor()
	if path.IsRoot() {
		n.state().Lock()
		n = node{rc: n.rc.AddRef()}
		n.state().Unlock()
	}

	// Iteratively "walk" to designated path.  It's important to avoid
	// recursion, so that RPCs can't blow up the stack.
	//
	// Each iteration of the loop shadows the n symbol, including its
	// embedded node, such that we are holding the final node when we
	// exit the loop.
	for path, name := path.Next(); name != ""; path, name = path.Next() {
		n = n.getOrCreateChild(name) // shallow copy
	}

	// Calling s.Anchor() transparently increments the reference count.
	return res.SetAnchor(n.Anchor())
}

func (n node) getOrCreateChild(name string) node {
	n.state().Lock()
	defer n.state().Unlock()

	state := n.state()

	if state.children == nil {
		state.children = make(map[string]*rc.Ref[nodestate])
	}

	// Fast path;  child exists
	if rc, ok := state.children[name]; ok {
		return node{
			rc: rc.AddRef(), // rule 2
		}
	}

	// Slow path; create new child.
	parent := n.rc.AddRef()
	state.children[name] = rc.NewRef(nodestate{}, func() { // rule 1
		defer parent.Release() // rule 3
		// defer n.rc.Release() // rule 3

		state.Lock()
		delete(state.children, name)
		state.Unlock()
	})

	return node{
		rc: state.children[name],
	}
}

func (n node) Anchor() api.Anchor {
	n.state().Lock()
	defer n.state().Unlock()

	// Fast path; a server is already running for this node.
	// Node is guaranteed to have r > 1 refs.  This means we
	// can release the refchain after client.AddRef returns.
	if n.state().client.Exists() {
		defer n.rc.Release() // rule 5 (Anchor)

		client := n.state().client.AddRef()
		return api.Anchor(client)
	}

	// Slow path; spin up a new server, assign the weak client,
	// and return the first reference.
	//
	// Node is guaranteed to have r > 0 refs. The refchain must
	// be released after the last client ref has been released.
	// This happens in the Shutdown() method, which is invoked
	// when the last client ref has been released.
	client := capnp.NewClient(&nodeRef{
		nodestate:  n.state(),
		ClientHook: api.Anchor_NewServer(n),
	})

	// Set the weak reference; subsequent calls to Anchor() will
	// derive clients from the weakref, incrementing the refcount.
	n.state().client = weakClient{
		WeakClient: client.WeakRef(),
		releaser:   n.rc.AddRef(),
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

func newPath(call api.Anchor_walk) Path {
	path, err := call.Args().Path()
	if err != nil {
		return failure(err)
	}

	return NewPath(path)
}

type nodestate struct {
	sync.Mutex
	children map[string]*rc.Ref[nodestate]
	client   weakClient
	// value api.Value
}

type weakClient struct {
	*capnp.WeakClient
	releaser *rc.Ref[nodestate]
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
	wc.releaser.Release()
	wc.releaser = nil
	wc.WeakClient = nil
}

func (wc weakClient) Exists() bool {
	return wc.WeakClient != nil
}

type nodeRef struct {
	*nodestate
	capnp.ClientHook
}

func (s *nodeRef) Shutdown() {
	s.Lock()
	defer s.Unlock()

	s.ClientHook.Shutdown()
}

// type Anchor api.Anchor

// func (a Anchor) AddRef() Anchor {
// 	return Anchor(api.Anchor(a).AddRef())
// }

// func (a Anchor) Release() {
// 	capnp.Client(a).Release()
// }

// func (a Anchor) Ls(ctx context.Context) (Iterator, capnp.ReleaseFunc) {
// 	f, release := api.Anchor(a).Ls(ctx, nil)

// 	h := &sequence{
// 		Future: casm.Future(f),
// 		pos:    -1,
// 	}

// 	return Iterator{
// 		Seq:    h,
// 		Future: h,
// 	}, release
// }

// // Walk to the register located at path.
// func (a Anchor) Walk(ctx context.Context, path string) (Anchor, capnp.ReleaseFunc) {
// 	p := NewPath(path)

// 	if p.IsRoot() {
// 		return Anchor(a), a.AddRef().Release
// 	}

// 	f, release := api.Anchor(a).Walk(ctx, destination(p))
// 	return Anchor(f.Anchor()), release
// }

// type Child interface {
// 	String() string
// 	Anchor() Anchor
// }

// type Iterator casm.Iterator[Child]

// func (it Iterator) Next() Child {
// 	r, _ := it.Seq.Next()
// 	return r
// }

// type sequence struct {
// 	casm.Future
// 	err error
// 	pos int
// }

// func (seq *sequence) Err() error {
// 	if seq.err == nil {
// 		select {
// 		case <-seq.Future.Done():
// 			_, seq.err = seq.Struct()
// 		default:
// 		}
// 	}

// 	return seq.err
// }

// func (seq *sequence) Next() (Child, bool) {
// 	if ok := seq.advance(); ok {
// 		return seq, true
// 	}

// 	return nil, false
// }

// func (seq *sequence) String() string {
// 	names, err := seq.results().Names()
// 	if err != nil {
// 		panic(err) // already validated; should never fail
// 	}

// 	name, err := names.At(seq.pos)
// 	if err != nil {
// 		panic(err)
// 	}

// 	return name
// }

// func (seq *sequence) Anchor() Anchor {
// 	children, err := seq.results().Children()
// 	if err != nil {
// 		panic(err) // already validated; should never fail
// 	}

// 	anchor, err := children.At(seq.pos)
// 	if err != nil {
// 		panic(err)
// 	}

// 	return Anchor(anchor)
// }

// func (seq *sequence) advance() (ok bool) {
// 	if ok = seq.Err() == nil; ok {
// 		seq.pos++
// 		ok = seq.validate()
// 	}

// 	return
// }

// func (seq *sequence) validate() bool {
// 	var s capnp.Struct
// 	if s, seq.err = seq.Struct(); seq.err != nil {
// 		return false
// 	}

// 	res := api.Anchor_ls_Results(s)
// 	if !(res.HasNames() && res.HasChildren()) {
// 		return false
// 	}

// 	return seq.validateName(res) && seq.validateChildren(res)
// }

// func (seq *sequence) validateName(res api.Anchor_ls_Results) (ok bool) {
// 	if ok = res.HasNames(); ok {
// 		names, err := res.Names()
// 		if ok = err == nil; ok {
// 			_, err = names.At(seq.pos)
// 			ok = err == nil
// 		}
// 	}

// 	return
// }

// func (seq *sequence) validateChildren(res api.Anchor_ls_Results) (ok bool) {
// 	if ok = res.HasChildren(); ok {
// 		children, err := res.Children()
// 		if ok = err == nil; ok {
// 			_, err = children.At(seq.pos)
// 			ok = err == nil
// 		}
// 	}

// 	return
// }

// func (seq *sequence) results() api.Anchor_ls_Results {
// 	res, err := seq.Struct()
// 	if err != nil {
// 		panic(err)
// 	}

// 	return api.Anchor_ls_Results(res)
// }

// func destination(path Path) func(api.Anchor_walk_Params) error {
// 	return func(ps api.Anchor_walk_Params) error {
// 		return path.bind(func(s string) bounded.Type[string] {
// 			err := ps.SetPath(trimmed(s))
// 			return bounded.Failure[string](err) // can be nil
// 		}).Err()
// 	}
// }
