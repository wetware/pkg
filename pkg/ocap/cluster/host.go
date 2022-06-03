package cluster

import (
	"context"
	"sync"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"capnproto.org/go/capnp/v3/server"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wetware/ww/internal/api/cluster"
	"github.com/wetware/ww/pkg/ocap/anchor"
	"github.com/wetware/ww/pkg/vat"
)

var HostCapability = vat.BasicCap{
	"host/packed",
	"host"}

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

func (h *Host) View(ctx context.Context, d Dialer) (FutureView, capnp.ReleaseFunc) {
	f, release := h.resolve(ctx, d).View(ctx, nil)
	return FutureView(f), release
}

func (h *Host) Ls(ctx context.Context, d Dialer) (*anchor.Iterator, capnp.ReleaseFunc) {
	return anchor.Anchor{Client: h.resolve(ctx, d).Client}.Ls(ctx)
}

// Walk to the register located at path.  Panics if len(path) == 0.
func (h *Host) Walk(ctx context.Context, d Dialer, path anchor.Path) (anchor.Anchor, capnp.ReleaseFunc) {
	return anchor.Anchor{Client: h.resolve(ctx, d).Client}.Walk(ctx, path)
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

type FutureView cluster.Host_view_Results_Future

func (f FutureView) Err() error {
	_, err := cluster.Host_view_Results_Future(f).Struct()
	return err
}

func (f FutureView) View() View {
	return View(cluster.Host_view_Results_Future(f).View())
}

func (f FutureView) Await(ctx context.Context) (View, error) {
	select {
	case <-f.Done():
		return f.View(), f.Err()
	case <-ctx.Done():
		return View{}, ctx.Err()
	}
}

/*---------------------------*
|                            |
|    Server Implementation   |
|                            |
*----------------------------*/

// HostServer represents a host instance on the network. It provides
// the Anchor and Joiner capabilities.
//
// The zero-value HostServer is ready to use.
type HostServer struct {
	once sync.Once

	// The server policy for client instances created by Client().
	// If nil, reasonable defaults are used. Callers MUST set this
	// value before the first call to Client().
	*server.Policy

	// RoutingTable provides a global view of namespace peers.
	// Callers MUST set this value before the first call to Client()
	RoutingTable

	// The root anchor for the HostServer.  Users SHOULD NOT
	// set this field; it will be populated automatically on
	// the first call to Client().
	anchor.AnchorServer
}

func (h *HostServer) Client() *capnp.Client {
	h.once.Do(func() {
		if h.AnchorServer.Path().IsZero() {
			h.AnchorServer = anchor.Root(h)
		}
	})

	return cluster.Host_ServerToClient(h, h.Policy).Client
}

func (h *HostServer) View(ctx context.Context, call cluster.Host_view) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	view := cluster.View_ServerToClient(ViewServer{
		RoutingTable: h.RoutingTable,
	}, h.Policy)

	return res.SetView(view)
}
