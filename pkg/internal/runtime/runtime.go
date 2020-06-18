package runtime

import (
	"context"

	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"

	log "github.com/lthibault/log/pkg"
	syncutil "github.com/lthibault/util/sync"

	"go.uber.org/fx"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-kad-dht/dual"

	"github.com/lthibault/wetware/pkg/discover"
	"github.com/lthibault/wetware/pkg/internal/p2p"
)

// HostEnv starts a runtime environment for a wetware host.
func HostEnv() fx.Option {
	return fx.Invoke(
		trackConnections,
		trackNeighbors,
		buildGraph,
		startBeacon,
		announce,
		listenAndJoin,
	)
}

// ClientEnv starts a runtime environment for a wetware client.
func ClientEnv() fx.Option {
	return fx.Invoke(
		trackConnections,
		dialAndJoin,
	)
}

type listenAndJoinParams struct {
	fx.In

	Log         log.Logger
	Host        host.Host
	ListenAddrs []multiaddr.Multiaddr
}

func listenAndJoin(ps listenAndJoinParams) error {
	ps.Log.Debug("listen and serve host")
	return ps.Host.(p2p.Listener).Listen(ps.ListenAddrs...)
}

type dialAndJoinParams struct {
	fx.In

	Log   log.Logger
	Host  host.Host
	Boot  discover.Strategy
	DHT   *dual.DHT `optional:"true"`
	Limit int       `name:"discover_limit" optional:"true"`
}

func dialAndJoin(ctx context.Context, ps dialAndJoinParams) error {
	peers, err := ps.Boot.DiscoverPeers(ctx,
		discover.WithLogger(ps.Log),
		discover.WithLimit(ps.Limit))
	if err != nil {
		return errors.Wrap(err, "discover")
	}

	any, ctxOK := syncutil.WithContext(ctx)
	go func() {
		for info := range peers {
			any.Go(connect(ctx, ps.Host, info))
		}
	}()

	select {
	case <-ctxOK.Done():
		// Best-effort attempt at booting the DHT, now that
		// a connection exists.
		//
		// This is a hacky attempt at fixing the kbucket.ErrLookupFailure when
		// invoking client commands.
		return ps.DHT.Bootstrap(ctx)
	case <-ctx.Done():
		return any.Wait()
	}
}

func connect(ctx context.Context, h host.Host, info peer.AddrInfo) func() error {
	return func() error {
		return h.Connect(ctx, info)
	}
}
