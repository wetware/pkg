package anchor

import (
	"sync"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/ww/internal/api/anchor"
	"zenhack.net/go/util/rc"
)

type node struct {
	rc *rc.Ref[nodestate]
}

func (n node) AddRef() node {
	return n.Bind(func(ref *rc.Ref[nodestate]) node {
		return node{rc: ref}
	})
}

func (n node) state() *nodestate {
	return n.rc.Value()
}

// Locking invariants are tricky, and using a functional style helps
// make them explicit. In essence, we must ensure that nodes are not
// removed during a traversal. To do so, we increase the refcount of
// each node along the traversal path. Because concurrent traversals
// may occur, we MUST hold the lock when incrementing the during the
// AddRef() operation. On the other hand, releasing a node may cause
// it to be removed from its parent's child-map, which in turn means
// that the parent may acquire its own lock when Release() is called.
//
// Thus, the tree exhibits bidirectional lock cascades; one starting
// at the root and flowing towards a leaf, and the other starting at
// an arbitrary node and flowing up towards the root.  When opposite
// cascades collide, it is essential that each hold only one lock at
// a time, else a deadlock will inevitably occur. Thus, we arrive at
// our three traversal invariants:
//
//  1. Traversals MUST increment the refcount of each node along
//     their paths.
//  2. A node's lock MUST be held when incrementing its refcount.
//  3. A node's lock MUST NOT be held when releasing its parent.
//
// Bind enforces these invariants during traversals. Release() logic
// is responsible for enforcing invariant #3.
func (n node) Bind(f func(*rc.Ref[nodestate]) node) node {
	n.state().Lock()
	defer n.state().Unlock()

	return f(n.rc.AddRef())
}

func child(name string) func(*rc.Ref[nodestate]) node {
	return func(parent *rc.Ref[nodestate]) node {
		state := parent.Value()

		// Fast path;  child exists.
		if child, ok := state.children[name]; ok {
			// The child holds a reference to the parent, so we are
			// certain that releasing the parent will not acquire a
			// lock.  This is safe.
			defer parent.Release()

			// A parent MUST NOT be released until its children are
			// all released; increment the child's refcount.
			return node{rc: child.AddRef()}
		}

		// Slow path; create new child.

		if state.children == nil {
			state.children = make(map[string]*rc.Ref[nodestate])
		}

		// The child holds the parent reference, releasing it when
		// its own refcount hit zero.
		state.children[name] = rc.NewRef(nodestate{}, func() {
			defer parent.Release()

			state.Lock()
			delete(state.children, name)
			state.Unlock()
		})

		return node{rc: state.children[name]}
	}
}

func (n node) Anchor() api.Anchor {
	n.state().Lock()
	defer n.state().Unlock()

	// Fast path; a server is already running for this node.
	// Node is guaranteed to have r > 1 refs.  This means we
	// can release the refchain after client.AddRef returns.
	if n.state().client.Exists() {
		defer n.rc.Release()

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
		ClientHook: api.Anchor_NewServer(server{n}),
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
