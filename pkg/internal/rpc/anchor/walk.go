package anchor

import (
	"context"

	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/lthibault/wetware/internal/api"
	ww "github.com/lthibault/wetware/pkg"
	anchorpath "github.com/lthibault/wetware/pkg/util/anchor/path"
	capnp "zombiezen.com/go/capnproto2"
)

type walk anchor

// Protocol ID
func (walk) Protocol() protocol.ID {
	return ww.Protocol
}

// HandleRPC for `walk`
func (h *walk) HandleRPC(ctx context.Context, c capnp.Client) {
	(*anchor)(h).anchorProvider = api.Anchor{Client: c}.
		Walk(ctx, func(p api.Anchor_walk_Params) error {
			return p.SetPath(anchorpath.Join(h.path))
		})
}
