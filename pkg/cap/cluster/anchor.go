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
}

func (h *Host) Join(ctx context.Context, d Dialer, peers []peer.AddrInfo) error {
	if len(peers) == 0 {
		return nil // nop
	}

	params := func(ps cluster.Host_join_Params) error {
		plist, err := ps.NewPeers(int32(len(peers)))
		if err != nil {
			return err
		}

		for i, info := range peers {
			if err = bindHostInfo(plist.At(i), info); err != nil {
				break
			}
		}

		return err
	}

	f, resolve := h.resolve(ctx, d).Join(ctx, params)
	defer resolve()

	_, err := f.Struct()
	return err
}

func (h *Host) Ls(ctx context.Context, d Dialer) (*RegisterMap, capnp.ReleaseFunc) {
	return listChildren(ctx, cluster.Anchor(h.resolve(ctx, d)))
}

// Walk to the register located at path.  Panics if len(path) == 0.
func (h *Host) Walk(ctx context.Context, d Dialer, path []string) (Register, capnp.ReleaseFunc) {
	return walkPath(ctx, cluster.Anchor(h.resolve(ctx, d)), path)
}

func (h *Host) resolve(ctx context.Context, d Dialer) cluster.Host {
	h.once.Do(func() {
		if h.Client == nil {
			if conn, err := d.Dial(ctx, h.Info); err != nil {
				h.Client = capnp.ErrorClient(err)
			} else {
				h.Client = conn.Bootstrap(ctx) // TODO:  wrap Client & call conn.Close() on Shutdown() hook?
			}
		}
	})

	return cluster.Host{Client: h.Client}
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

// MergeStrategy is responsible for merging two disjoing clusters
// in the same namespace.
type MergeStrategy interface {
	// Merge the host cluster with the cluster to which each peer
	// belongs. If peer is already a member of the local cluster,
	// Merge is effectively a nop.  Note that implementations MAY
	// nevertheless exhibit side-effects in such cases, typically
	// in the form of network requests and DHT refreshes.
	//
	// Implementations MUST ensure that members of peers' cluster
	// appear in the local host's routing table after the call to
	// Merge returns.  They SHOULD also ensure that services such
	// as the DHT are refreshed, as needed, and block until these
	// operations are complete.
	Merge(ctx context.Context, peers []peer.AddrInfo) error
}

type HostServer struct {
	node    node
	cluster MergeStrategy
}

func NewHost(m MergeStrategy) HostServer {
	s := HostServer{
		cluster: m,
		node: node{
			cs: make(map[string]node),
			mu: new(sync.RWMutex),
		},
	}

	s.node.Anchor.Client = cluster.Host_ServerToClient(s, &defaultPolicy).Client
	return s
}

func (s HostServer) Client() *capnp.Client { return s.node.Anchor.Client }

func (s HostServer) Join(ctx context.Context, call cluster.Host_join) error {
	ps, err := call.Args().Peers()
	if err != nil {
		return err
	}

	var peers = make([]peer.AddrInfo, ps.Len())
	for i := range peers {
		if err = bindAddrInfo(&peers[i], ps.At(i)); err != nil {
			return err
		}
	}

	return s.cluster.Merge(ctx, peers)
}

func (s HostServer) Ls(ctx context.Context, call cluster.Anchor_ls) error {
	return s.node.Ls(ctx, call)
}

func (s HostServer) Walk(ctx context.Context, call cluster.Anchor_walk) error {
	return s.node.Walk(ctx, call)
}

// node is theserver implemenation for host-local Anchors.
type node struct {
	Name   string
	Anchor cluster.Anchor // client capability for node

	mu     *sync.RWMutex
	parent *node
	cs     map[string]node
	// value  interface{}
}

func (n node) Shutdown() {
	defer n.parent.Release()

	n.parent.mu.Lock()
	defer n.parent.mu.Unlock()

	delete(n.parent.cs, n.Name)
}

func (n node) Release() { n.Anchor.Release() }

func (n node) AddRef() *node {
	return &node{
		Name:   n.Name,
		Anchor: n.Anchor.AddRef(),
		parent: n.parent,
		cs:     n.cs,
		mu:     n.mu,
	}
}

func (n node) Ls(_ context.Context, call cluster.Anchor_ls) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	n.mu.RLock()
	defer n.mu.RUnlock()

	cs, err := res.NewChildren(int32(len(n.cs)))
	if err != nil {
		return err
	}

	var i int
	for name, n := range n.cs {
		if err = cs.At(i).SetName(name); err != nil {
			return err
		}

		if err = cs.At(i).SetAnchor(n.Anchor); err != nil {
			return err
		}
	}

	return nil
}

func (n node) Walk(ctx context.Context, call cluster.Anchor_walk) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	path, err := call.Args().Path()
	if err != nil {
		return err
	}

	c, err := n.walk(path)
	if err != nil {
		return err
	}

	return res.SetAnchor(c.Anchor)
}

func (n node) walk(path capnp.TextList) (*node, error) {
	for i := 0; i < path.Len(); i++ {
		name, err := path.At(i)
		if err != nil {
			return nil, err
		}

		n = n.child(name)
	}

	// BUG:  n's reference may have hit zero in the meantime,
	//       which will cause a panic.  (Lock is held in n.child)
	return n.AddRef(), nil
}

func (n node) child(name string) node {
	n.mu.Lock()
	defer n.mu.Unlock()

	if c, ok := n.cs[name]; ok {
		return c
	}

	// slow path - create new node
	c := node{
		Name:   name,
		parent: n.AddRef(),
		cs:     make(map[string]node),
		mu:     new(sync.RWMutex),
	}

	c.Anchor = cluster.Anchor_ServerToClient(c, &defaultPolicy)
	n.cs[name] = c

	return c // ref = 1
}
