package auth

import (
	"context"
	"fmt"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p/core/peer"

	api "github.com/wetware/pkg/api/cluster"
)

type Policy func(context.Context, api.Host, peer.ID) (api.Host, capnp.ReleaseFunc)

func AllowAll(_ context.Context, root api.Host, _ peer.ID) (api.Host, capnp.ReleaseFunc) {
	return api.Host(root), root.Release
}

func Deny(reason string, args ...any) Policy {
	return func(context.Context, api.Host, peer.ID) (api.Host, capnp.ReleaseFunc) {
		err := fmt.Errorf(reason, args...)
		client := capnp.ErrorClient(err)
		return api.Host(client), func() {}
	}
}

func (auth Policy) Authenticate(ctx context.Context, root api.Host, account peer.ID) (api.Host, capnp.ReleaseFunc) {
	if auth == nil {
		// Default to denying all auth requests.
		return Deny("no policy").Authenticate(ctx, root, account)
	}

	return auth(ctx, root, account)
}
