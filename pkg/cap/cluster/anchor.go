package cluster

import (
	"context"
	"sync"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wetware/ww/internal/api/cluster"
	"github.com/wetware/ww/pkg/vat"
)

var AnchorCapability = vat.BasicCap{
	"anchor/packed",
	"anchor"}

/*----------------------------*
|                             |
|    Client Implementations   |
|                             |
*-----------------------------*/
type Dialer interface {
	Dial(context.Context, peer.AddrInfo) (*rpc.Conn, error)
}

type Host struct {
	once   sync.Once
	Client *capnp.Client

	Info   peer.AddrInfo
	Dialer Dialer
}

func (h *Host) Ls(ctx context.Context) (*RegisterMap, capnp.ReleaseFunc) {
	return listChildren(ctx, h.resolve(ctx))
}

// Walk to the register located at path.  Panics if len(path) == 0.
func (h *Host) Walk(ctx context.Context, path []string) (Register, capnp.ReleaseFunc) {
	return walkPath(ctx, h.resolve(ctx), path)
}

func (h *Host) resolve(ctx context.Context) cluster.Anchor {
	h.once.Do(func() {
		if h.Client == nil {
			if conn, err := h.Dialer.Dial(ctx, h.Info); err != nil {
				h.Client = capnp.ErrorClient(err)
			} else {
				h.Client = conn.Bootstrap(ctx) // TODO:  wrap Client & call conn.Close() on Shutdown() hook?
			}
		}
	})

	return cluster.Anchor{Client: h.Client}
}

type RegisterMap struct {
	Err  error
	Name string
	pos  int
	cs   cluster.Anchor_Child_List
}

func regmap(cs cluster.Anchor_Child_List) *RegisterMap {
	return &RegisterMap{cs: cs}
}

func errmap(err error) *RegisterMap {
	return &RegisterMap{Err: err}
}

func (rs *RegisterMap) More() bool {
	return rs.Err == nil && rs.pos < rs.cs.Len()
}

func (rs *RegisterMap) Next() (more bool) {
	if more = rs.More(); more {
		rs.Name, rs.Err = rs.cs.At(rs.pos).Name()
		rs.pos++
	}

	return
}

func (rs *RegisterMap) Register() Register {
	return Register(rs.cs.At(rs.pos).Anchor())
}

type Register cluster.Anchor

func (r Register) Ls(ctx context.Context) (*RegisterMap, capnp.ReleaseFunc) {
	return listChildren(ctx, cluster.Anchor(r))
}

// Walk to the register located at path.  Panics if len(path) == 0.
func (r Register) Walk(ctx context.Context, path []string) (Register, capnp.ReleaseFunc) {
	return walkPath(ctx, cluster.Anchor(r), path)
}

func (r Register) AddRef() Register {
	return Register(cluster.Anchor(r) /*.AddRef()*/)
}

/*

	Generic methods for client implementations

*/

func listChildren(ctx context.Context, a cluster.Anchor) (*RegisterMap, capnp.ReleaseFunc) {
	f, release := a.Ls(ctx, nil)

	res, err := f.Struct()
	if err != nil {
		release()
		return errmap(err), func() {}
	}

	cs, err := res.Children()
	if err != nil {
		release()
		return errmap(err), func() {}
	}

	return regmap(cs), release
}

func walkPath(ctx context.Context, a cluster.Anchor, path []string) (Register, capnp.ReleaseFunc) {
	if len(path) == 0 {
		// While not strictly necessary, requiring non-empty paths
		// simplifies the ref-counting logic considerably.  Nop walks
		// are implemented by client.register, which wraps this type.
		panic("zero-length path")
	}

	f, release := a.Walk(ctx, walkParam(path))
	return Register(f.Anchor()), release
}

func walkParam(path []string) func(cluster.Anchor_walk_Params) error {
	return func(ps cluster.Anchor_walk_Params) error {
		p, err := ps.NewPath(int32(len(path)))
		if err == nil {
			for i, e := range path {
				if err = p.Set(i, e); err != nil {
					break
				}
			}
		}

		return err
	}
}

/*----------------------------*
|                             |
|    Server Implementations   |
|                             |
*-----------------------------*/

type HostServer struct {
	mu       sync.RWMutex
	children nodeMap
	anchor   cluster.Anchor
}

func NewHost() *HostServer {
	s := &HostServer{children: make(map[string]node)}
	s.anchor = cluster.Anchor_ServerToClient(s, &defaultPolicy)
	return s
}

func (s *HostServer) Client() *capnp.Client {
	return s.anchor.Client
}

func (s *HostServer) Ls(_ context.Context, call cluster.Anchor_ls) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.children.HandleLs(call)
}

func (s *HostServer) Walk(_ context.Context, call cluster.Anchor_walk) error {
	return walkHandler{
		Lock:     &s.mu,
		Parent:   nothing(),
		Anchor:   s.anchor,
		Children: s.children,
	}.ServeRPC(call)
}

// node is theserver implemenation for host-local Anchors.
type node struct {
	Name   string
	Anchor cluster.Anchor // client capability for node

	mu       *sync.RWMutex
	parent   maybeParent
	children nodeMap
	value    interface{}
}

func (n node) Shutdown() {
	// BUG:  This isn't getting called for Host's immediate children
	//       because their parent is nil.
	n.parent.Bind(func(p *node) maybeParent {
		defer p.Release()

		// We MUST release the lock before calling release, else the
		// entire path will be locked.  If a concurrent call to walk
		// were to descend the same path, a deadlock would occur.
		p.mu.Lock()
		defer p.mu.Unlock()

		delete(p.children, n.Name)

		return nothing()
	})
}

func (n node) AddRef() node {
	return node{
		Name:     n.Name,
		value:    n.value,
		Anchor:   n.Anchor.AddRef(),
		parent:   n.parent,
		mu:       n.mu,
		children: n.children,
	}
}

func (n node) Release() {
	n.Anchor.Release() // nil-safe
}

func (n node) Lock() {
	if n.mu != nil {
		n.mu.Lock()
	}
}

func (n node) Unlock() {
	if n.mu != nil {
		n.mu.Unlock()
	}
}

func (n node) Path() (path []string) {
	n.parent.Bind(func(p *node) maybeParent {
		path = p.Path()
		return nothing()
	})

	return append(path, n.Name)
}

func (n node) Ls(_ context.Context, call cluster.Anchor_ls) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.children.HandleLs(call)
}

func (n node) Walk(ctx context.Context, call cluster.Anchor_walk) error {
	return walkHandler{
		Lock:     n.mu,
		Parent:   n.parent,
		Anchor:   n.Anchor,
		Children: n.children,
	}.ServeRPC(call)
}

// nodeMap is a generic mapping of names to child nodes.  It is used by
// both Server and node.
type nodeMap map[string]node

func (m nodeMap) GetOrCreate(name string, parent maybeParent) node {
	if n, ok := m[name]; ok {
		return n
	}

	// slow path - create new node
	n := node{
		Name:     name,
		mu:       new(sync.RWMutex),
		parent:   parent,
		children: make(nodeMap),
	}

	n.Anchor = cluster.Anchor_ServerToClient(n, &defaultPolicy)

	m[name] = n
	return n // ref = 1
}

// maybeParent is a simple sum type for parent references, which are
// nullable.
type maybeParent struct{ *node }

func maybe(n *node) maybeParent {
	return maybeParent{n}
}

func just(n *node) maybeParent {
	if n == nil {
		panic("just(nil)")
	}

	return maybeParent{n}
}

func nothing() maybeParent { return maybeParent{} }

func (p maybeParent) Bind(f func(n *node) maybeParent) maybeParent {
	if p.node == nil {
		return nothing()
	}

	return f(p.node)
}

// func addref(n *node) maybeParent {
// 	n.mu.RLock()
// 	defer n.mu.RUnlock()

// 	u := n.AddRef()
// 	return just(&u)
// }

/*

	Generic method handlers for all server implementations

*/

func (m nodeMap) HandleLs(call cluster.Anchor_ls) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	cs, err := res.NewChildren(int32(len(m)))
	if err != nil {
		return err
	}

	var i int
	for name, n := range m {
		if err = cs.At(i).SetName(name); err != nil {
			return err
		}

		if err = cs.At(i).SetAnchor(n.Anchor.AddRef()); err != nil {
			return err
		}
	}

	return nil
}

type walkHandler struct {
	Lock     sync.Locker
	Parent   maybeParent
	Anchor   cluster.Anchor
	Children nodeMap
}

func (h walkHandler) ServeRPC(call cluster.Anchor_walk) error {
	path, err := call.Args().Path()
	if err != nil {
		return err
	}

	return h.visitor(path).BindWalk(call)
}

func (h walkHandler) visitor(path capnp.TextList) visitor {
	// This is the only stateful part of walkHandler. We
	// update Parent/Anchor/Children in a loop until the
	// destination is reached or an error is encountered.
	for i := 0; i < path.Len(); i++ {
		name, err := path.At(i)
		if err != nil {
			return abort(err) // TODO:  release parent
		}

		h = h.step(name)
	}

	return jump(h.Anchor)
}

func (h walkHandler) step(name string) walkHandler {
	h.Lock.Lock()
	defer h.Lock.Unlock()

	n := h.Children.GetOrCreate(name, h.Parent)

	return walkHandler{
		Lock:     n.mu,
		Parent:   n.parent,
		Anchor:   n.Anchor,
		Children: n.children,
	}
}

type visitor func(func(cluster.Anchor) error) error

func jump(a cluster.Anchor) visitor {
	return func(visit func(cluster.Anchor) error) error {
		return visit(a.AddRef())
	}
}

func abort(err error) visitor {
	return func(func(cluster.Anchor) error) error {
		return err
	}
}

func (bind visitor) BindWalk(call cluster.Anchor_walk) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return bind(res.SetAnchor)
}
