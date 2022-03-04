package runtime

import (
	"context"
	"os"
	"path/filepath"

	"github.com/lthibault/log"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	ds "github.com/ipfs/go-datastore"
	badgerds "github.com/ipfs/go-ds-badger2"

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

func storage(c *cli.Context, log log.Logger, lx fx.Lifecycle) ds.Batching {
	path, err := homedir.Expand(c.Path("data"))
	if err != nil {
		log.WithField("data_dir", c.Path("data")).Fatal(err)
	}

	path = filepath.Join(path, "data")

	if err = os.MkdirAll(path, 0700); err != nil {
		log.Fatal(err)
	}

	log = log.WithField("data_dir", path)
	badgerds.DefaultOptions.Logger = badgerLogger{log}

	d, err := badgerds.NewDatastore(
		filepath.Join(path, "data"),
		&badgerds.DefaultOptions)
	if err != nil {
		log.Fatalf("badgerdb: %s", err)
	}

	lx.Append(closer(d))
	lx.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			log.Trace("syncing datastore")
			return d.Sync(ctx, ds.NewKey("/"))
		},
	})

	return d
}

type badgerLogger struct{ log.Logger }

func (b badgerLogger) Warningf(fmt string, vs ...interface{}) {
	b.Warnf(fmt, vs...)
}
