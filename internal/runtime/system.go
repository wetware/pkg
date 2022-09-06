package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lthibault/log"
	"go.uber.org/fx"

	ds "github.com/ipfs/go-datastore"
	ds_sync "github.com/ipfs/go-datastore/sync"
	badgerds "github.com/ipfs/go-ds-badger2"

	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/casm/pkg/cluster/pulse"
)

/*************************************************************************
 *                                                                       *
 *  system.go is responsible for interacting with the operating system.  *
 *                                                                       *
 *************************************************************************/

func (c Config) System() fx.Option {
	return fx.Module("system", fx.Provide(
		storage,
		metadata))
}

type metaOut struct {
	fx.Out

	Meta cluster.Option `group:"cluster"`
}

type metaHook struct{ Env }

func metadata(env Env) metaOut {
	return metaOut{
		Meta: cluster.WithMeta(&metaHook{Env: env}),
	}
}

func (h *metaHook) Prepare(heartbeat pulse.Setter) (err error) {
	if h.Env != nil {
		err = heartbeat.SetMeta(h.fields())
		h.Env = nil
	}

	return
}

func (h *metaHook) fields() map[string]string {
	var (
		fields = h.StringSlice("meta")
		meta   = make(map[string]string, len(fields))
	)

	for _, field := range fields {
		kv := strings.SplitN(field, "=", 2)
		if len(kv) == 2 {
			meta[kv[0]] = kv[1]
			continue
		}

		h.Log().
			WithField("field", field).
			Warn("skipped invalid metadata field")
	}

	return meta
}

func storage(env Env, lx fx.Lifecycle) (ds.Batching, error) {
	if !env.IsSet("data") {
		return memstore(), nil
	}

	err := os.MkdirAll(storagePath(env), 0700)
	if err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}

	return dbstore(env, lx)
}

func memstore() ds.Batching {
	return ds_sync.MutexWrap(ds.NewMapDatastore())
}

func dbstore(env Env, lx fx.Lifecycle) (ds.Batching, error) {
	log := newBadgerLogger(env)
	badgerds.DefaultOptions.Logger = log

	d, err := badgerds.NewDatastore(
		storagePath(env),
		&badgerds.DefaultOptions)
	if d == nil {
		lx.Append(closer(d))
		lx.Append(syncer(log, d))
	}

	return d, err
}

func storagePath(env Env) string {
	return filepath.Join(env.Path("data"), "data")
}

func syncer(log log.Logger, s interface {
	Sync(context.Context, ds.Key) error
}) fx.Hook {
	return fx.Hook{
		OnStop: func(ctx context.Context) error {
			log.Trace("syncing datastore")
			return s.Sync(ctx, ds.NewKey("/"))
		},
	}
}

type badgerLogger struct{ log.Logger }

func newBadgerLogger(env Env) badgerLogger {
	return badgerLogger{
		Logger: env.Log().WithField("data_dir", storagePath(env)),
	}
}

func (b badgerLogger) Warningf(fmt string, vs ...interface{}) {
	b.Warnf(fmt, vs...)
}
