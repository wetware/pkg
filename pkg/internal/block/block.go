// Package block implements efficient storage and transfer of arbitrary blocks of data.
package block

/*
	TODO(refactor):  package name `block` is easily confused with github.com/ipfs/go-block-format.
*/

import (
	"context"
	"io"

	"go.uber.org/fx"

	"github.com/ipfs/go-bitswap"
	"github.com/ipfs/go-bitswap/network"
	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-datastore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	exchange "github.com/ipfs/go-ipfs-exchange-interface"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
)

// Config for block services.
type Config struct {
	fx.In

	Host host.Host
	DHT  routing.Routing

	/*
		TODO(performance):  pass in some sort of persistent implementation (badgerdb?)

		Currently there is a single memory-backed datastore that gets passed in here.
		Investigate the possibility of maintaining two separate datastores:

			- MapDatastore for DHT & other volatile data
			- Persistent datastore for block data

		Resist the urge to optimize this prematurely.  It's unclear whether this will
		work without heavy modification to IPFS.  (e.g.:  what happens after a restart?
		Will the DHT be automatically populated from the bitswap exchange?)
	*/
	Store datastore.Batching
}

// Module contains primitives for working with blocks of data.
type Module struct {
	fx.Out

	Service  blockservice.BlockService
	GCLocker blockstore.GCLocker
}

// New .
func New(ctx context.Context, cfg Config, lx fx.Lifecycle) (mod Module, err error) {
	bs := blockstore.NewBlockstore(cfg.Store)

	/*
		TODO(performance): investigate persistent blockstore with caching (see below)
	*/
	// if bs, err = blockstore.CachedBlockstore(ctx, bs, blockstore.DefaultCacheOpts()); err != nil {
	// 	return
	// }

	mod.GCLocker = blockstore.NewGCLocker()
	exc := bitswap.New(ctx,
		network.NewFromIpfsHost(cfg.Host, cfg.DHT),
		blockstore.NewGCBlockstore(bs, mod.GCLocker),
		bitswap.ProvideEnabled(true),
	).(exchange.SessionExchange)
	lx.Append(closehook(exc))

	mod.Service = blockservice.New(bs, exc)

	return
}

func closehook(c io.Closer) fx.Hook {
	return fx.Hook{
		OnStop: func(context.Context) error {
			return c.Close()
		},
	}
}
