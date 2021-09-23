package start

import (
	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"

	// badger "github.com/ipfs/go-ds-badger2"
	"github.com/urfave/cli/v2"
)

func newDatastore(c *cli.Context) (ds.Batching, error) {
	// TODO(enhancement):  use peristent datastore + namespacing + caching.
	return sync.MutexWrap(ds.NewMapDatastore()), nil
	// return badger.NewDatastore(c.Path("store"), &badger.DefaultOptions)
}
