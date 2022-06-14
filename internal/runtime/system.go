package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lthibault/log"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"
	"gopkg.in/alexcesaro/statsd.v2"

	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	badgerds "github.com/ipfs/go-ds-badger2"

	"github.com/wetware/casm/pkg/cluster/pulse"
	logutil "github.com/wetware/ww/internal/util/log"
	statsdutil "github.com/wetware/ww/internal/util/statsd"
)

/*******************************************************************************
 *                                                                             *
 *  system.go is responsible for interacting with the local operating system.  *
 *                                                                             *
 *******************************************************************************/

// system module:  interacts with local file storage.
var system = fx.Options(
	observability,
	fx.Provide(
		storage,
		heartbeat))

var observability = fx.Provide(
	logging,
	statsdutil.New,
	statsdutil.NewBandwidthCounter,
	statsdutil.NewPubSubTracer,
	NewWwMetricsReporter)

func logging(c *cli.Context) log.Logger {
	return logutil.New(c).With(log.F{
		"ns": c.String("ns"),
	})
}

func NewWwMetricsReporter(c *cli.Context, client *statsd.Client) *statsdutil.MetricsReporter {
	metrics := statsdutil.NewMetricsReporter(client)
	go metrics.Run(c.Context)
	return metrics
}

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

type storageConfig struct {
	fx.In

	CLI *cli.Context
	Log log.Logger

	Lifecycle fx.Lifecycle
}

func (config storageConfig) IsVolatile() bool {
	return !config.CLI.IsSet("data")
}

func (config storageConfig) StoragePath() string {
	return filepath.Join(config.CLI.Path("data"), "data")
}

func (config storageConfig) Logger() badgerLogger {
	if config.IsVolatile() {
		return badgerLogger{config.Log}
	}

	return badgerLogger{
		config.Log.WithField("data_dir", config.StoragePath()),
	}
}

func (config storageConfig) VolatileStorage() ds.Batching {
	return sync.MutexWrap(ds.NewMapDatastore())
}

func (config storageConfig) PersistentStorage() (ds.Batching, error) {
	err := os.MkdirAll(config.StoragePath(), 0700)
	if err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}

	badgerds.DefaultOptions.Logger = config.Logger()

	return badgerds.NewDatastore(
		config.StoragePath(),
		&badgerds.DefaultOptions)
}

func (config storageConfig) SyncOnClose(d ds.Datastore) {
	config.Lifecycle.Append(closer(d))
	config.Lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			log.Trace("syncing datastore")
			return d.Sync(ctx, ds.NewKey("/"))
		},
	})
}

func storage(config storageConfig) (ds.Batching, error) {
	if config.IsVolatile() {
		return config.VolatileStorage(), nil
	}

	d, err := config.PersistentStorage()
	if err == nil {
		config.SyncOnClose(d)
	}

	return d, err
}

type badgerLogger struct{ log.Logger }

func (b badgerLogger) Warningf(fmt string, vs ...interface{}) {
	b.Warnf(fmt, vs...)
}
