package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lthibault/log"
	"github.com/wetware/casm/pkg/cluster/pulse"
	"github.com/wetware/casm/pkg/cluster/routing"
	"go.uber.org/fx"

	ds "github.com/ipfs/go-datastore"
	ds_sync "github.com/ipfs/go-datastore/sync"
	badgerds "github.com/ipfs/go-ds-badger2"
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

type meta struct {
	fields []routing.Field
}

func metadata(env Env) (pulse.Preparer, error) {
	ss := env.StringSlice("meta")
	fs := make([]routing.Field, len(ss))

	var err error
	for i, field := range ss {
		if fs[i], err = routing.ParseField(field); err != nil {
			return nil, err
		}
	}

	return &meta{
		fields: fs,
	}, nil
}

func (m *meta) Prepare(h pulse.Heartbeat) error {
	// write meta fields only once
	if len(m.fields) > 0 {
		if err := h.SetMeta(m.fields); err != nil {
			return err
		}

		m.fields = nil
	}

	// hostname may change over time
	host, err := os.Hostname()
	if err != nil {
		return err
	}

	return h.SetHost(host)

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
