package anchor

import (
	"context"
	"sync"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wetware/ww/internal/api/anchor"
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

	params := func(ps anchor.Host_join_Params) error {
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
	return listChildren(ctx, anchor.Anchor(h.resolve(ctx, d)))
}

// Walk to the register located at path.  Panics if len(path) == 0.
func (h *Host) Walk(ctx context.Context, d Dialer, path Path) (Register, capnp.ReleaseFunc) {
	child := anchor.Anchor(h.resolve(ctx, d))
	return walkPath(ctx, child, path)
}

func (h *Host) resolve(ctx context.Context, d Dialer) anchor.Host {
	h.once.Do(func() {
		if h.Client == nil {
			if conn, err := d.Dial(ctx, h.Info); err != nil {
				h.Client = capnp.ErrorClient(err)
			} else {
				h.Client = conn.Bootstrap(ctx) // TODO:  wrap Client & call conn.Close() on Shutdown() hook?
			}
		}
	})

	return anchor.Host{Client: h.Client}
}

type RegisterMap struct {
	Err  error
	Name string
	pos  int
	cs   anchor.Anchor_Child_List
}

func regmap(cs anchor.Anchor_Child_List) *RegisterMap {
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

type Register anchor.Anchor

func (r Register) Ls(ctx context.Context) (*RegisterMap, capnp.ReleaseFunc) {
	return listChildren(ctx, anchor.Anchor(r))
}

// Walk to the register located at path.  Panics if len(path) == 0.
func (r Register) Walk(ctx context.Context, path string) (Register, capnp.ReleaseFunc) {
	return walkPath(ctx, anchor.Anchor(r), NewPath(path))
}

func (r Register) AddRef() Register {
	return Register(anchor.Anchor(r) /*.AddRef()*/)
}

/*

	Generic methods for client implementations

*/

func listChildren(ctx context.Context, a anchor.Anchor) (*RegisterMap, capnp.ReleaseFunc) {
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

func walkPath(ctx context.Context, a anchor.Anchor, path Path) (Register, capnp.ReleaseFunc) {
	if path.String() == "" {
		// While not strictly necessary, requiring non-empty paths
		// simplifies the ref-counting logic considerably.  Nop walks
		// are implemented by client.register, which wraps this type.
		panic("empty path")
	}

	f, release := a.Walk(ctx, walkParam(path))
	return Register(f.Anchor()), release
}

func walkParam(path Path) func(anchor.Anchor_walk_Params) error {
	return func(ps anchor.Anchor_walk_Params) error {
		return path.Bind(ps)
	}
}

/*---------------------------*
|                            |
|    Server Implementation   |
|                            |
*----------------------------*/

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
	MergeStrategy
	Anchor
}

func NewHost(m MergeStrategy) *HostServer {
	h := &HostServer{MergeStrategy: m}
	h.Anchor = Root(h)
	return h
}

func (h *HostServer) Join(ctx context.Context, call anchor.Host_join) error {
	ps, err := call.Args().Peers()
	if err != nil {
		return err
	}

	// TODO:  use an interface to pass ps to Merge() directly, rather than
	//        allocating a slice.
	var peers = make([]peer.AddrInfo, ps.Len())
	for i := range peers {
		if err = bindAddrInfo(&peers[i], ps.At(i)); err != nil {
			return err
		}
	}

	return h.Merge(ctx, peers)
}
