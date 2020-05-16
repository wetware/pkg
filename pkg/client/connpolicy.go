package client

import (
	"context"

	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	log "github.com/lthibault/log/pkg"
	syncutil "github.com/lthibault/util/sync"
	discover "github.com/lthibault/wetware/pkg/discover"
	"github.com/pkg/errors"
	"go.uber.org/fx"
)

/*
	connpolicy.go contains the logic responsible for ensuring a client stays connected
	to a cluster.
*/

type dialConfig struct {
	fx.In

	Ctx  context.Context
	Log  log.Logger
	Host host.Host

	discover.Strategy
	Limit int `name:"discover_limit"`
}

// dialer attempts to connect to n peers, returning when we have at least one successful
// connection.
func dialer(ctx context.Context) func(fx.Lifecycle, dialConfig) error {
	return func(lx fx.Lifecycle, cfg dialConfig) error {
		ps, err := cfg.DiscoverPeers(ctx,
			discover.WithLogger(cfg.Log),
			discover.WithLimit(cfg.Limit))
		if err != nil {
			return errors.Wrap(err, "discover")
		}

		any, ctxOK := syncutil.WithContext(ctx)
		go func() {
			for info := range ps {
				any.Go(connect(ctx, cfg.Host, info))
			}
		}()

		select {
		case <-ctxOK.Done():
			return nil
		case <-ctx.Done():
			return any.Wait()
		}
	}
}

func connect(ctx context.Context, h host.Host, info peer.AddrInfo) func() error {
	return func() error {
		return h.Connect(ctx, info)
	}
}
