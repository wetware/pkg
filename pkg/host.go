//go:generate mockgen -source=host.go -destination=../internal/mock/pkg/host.go -package=mock_ww

package ww

import (
	"context"

	"capnproto.org/go/capnp/v3"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/cluster"
	api "github.com/wetware/ww/internal/api/cluster"
)

var HostCapability = casm.BasicCap{
	"host/packed",
	"host"}

/*----------------------------*
|                             |
|    Client Implementations   |
|                             |
*-----------------------------*/

// type Dialer interface {
// 	Dial(context.Context, peer.AddrInfo) (*rpc.Conn, error)
// }

type Host api.Host

func (h Host) AddRef() Host {
	return Host(capnp.Client(h).AddRef())
}

func (h Host) Release() {
	capnp.Client(h).Release()
}

func (h Host) View(ctx context.Context) (cluster.View, capnp.ReleaseFunc) {
	f, release := api.Host(h).View(ctx, nil)
	return cluster.View(f.View().Client()), release
}

// func (h *Host) Ls(ctx context.Context, d Dialer) (*anchor.Iterator, capnp.ReleaseFunc) {
// 	return anchor.Anchor(h.resolve(ctx, d)).Ls(ctx)
// }

// // Walk to the register located at path.  Panics if len(path) == 0.
// func (h *Host) Walk(ctx context.Context, d Dialer, path anchor.Path) (anchor.Anchor, capnp.ReleaseFunc) {
// 	return anchor.Anchor(h.resolve(ctx, d)).Walk(ctx, path)
// }

// func (h *Host) resolve(ctx context.Context, d Dialer) api.Host {
// 	h.once.Do(func() {
// 		if h.Client == (capnp.Client{}) {
// 			if conn, err := d.Dial(ctx, h.Info); err != nil {
// 				h.Client = capnp.ErrorClient(err)
// 			} else {
// 				h.Client = conn.Bootstrap(ctx) // TODO:  wrap Client & call conn.Close() on Shutdown() hook?
// 			}
// 		}
// 	})

// 	return api.Host(h.Client)
// }

/*---------------------------*
|                            |
|    Server Implementation   |
|                            |
*----------------------------*/

type ViewProvider interface {
	View() cluster.View
}

// HostServer provides the Host capability.
type HostServer struct {
	ViewProvider ViewProvider
}

func (s HostServer) Client() capnp.Client {
	return capnp.Client(s.Host())
}

func (s HostServer) Host() Host {
	return Host(api.Host_ServerToClient(s))
}

func (s HostServer) View(_ context.Context, call api.Host_view) error {
	res, err := call.AllocResults()
	if err == nil {
		err = res.SetView(capnp.Client(s.ViewProvider.View()))
	}

	return err
}
