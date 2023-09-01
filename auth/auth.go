package auth

import (
	"context"
	"fmt"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p/core/peer"

	api "github.com/wetware/pkg/api/cluster"
	"github.com/wetware/pkg/cap/host"
)

type Policy func(context.Context, host.Host, peer.ID) (api.Host, capnp.ReleaseFunc)

func AllowAll(_ context.Context, root host.Host, _ peer.ID) (api.Host, capnp.ReleaseFunc) {
	return api.Host(root), root.Release
}

func Deny(reason string, args ...any) Policy {
	return func(context.Context, host.Host, peer.ID) (api.Host, capnp.ReleaseFunc) {
		err := fmt.Errorf(reason, args...)
		client := capnp.ErrorClient(err)
		return api.Host(client), func() {}
	}
}

func (auth Policy) Authenticate(ctx context.Context, root host.Host, account peer.ID) (api.Host, capnp.ReleaseFunc) {
	if auth == nil {
		// Default to denying all auth requests.
		return Deny("no policy").Authenticate(ctx, root, account)
	}

	return auth(ctx, root, account)
}

func (auth Policy) Export(h host.Host) Terminal {
	return Terminal(api.Terminal_ServerToClient(TerminalServer{
		Host:   h,
		Policy: auth,
	}))
}
