package runtime

import (
	"go.uber.org/fx"

	// badger "github.com/ipfs/go-ds-badger2"
	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"

	"github.com/wetware/casm/pkg/cluster/pulse"
)

// system module:  interacts with local file storage.
var system = fx.Provide(
	storage,
	heartbeat,
)

/*******************************************************************************
 *                                                                             *
 *  system.go is responsible for interacting with the local operating system.  *
 *                                                                             *
 *******************************************************************************/

// hook populates heartbeat messages with system information from the
// operating system.
type hook struct{}

func heartbeat() hook { return hook{} }

func (h hook) Prepare(pulse.Heartbeat) {
	// TODO:  populate a capnp struct containing metadata for the
	//        local host.  Consider including AWS AR information,
	//        hostname, geolocalization, and a UUID instance id.

	// WARNING:  DO NOT make a syscall each time 'Prepare' is invoked.
	//           Cache results and periodically refresh them.
}

func storage() ds.Batching {
	// TODO(enhancement):  use peristent datastore + namespacing + caching.
	return sync.MutexWrap(ds.NewMapDatastore())
	// return badger.NewDatastore(c.Path("store"), &badger.DefaultOptions)
}
