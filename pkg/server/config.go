package server

import (
	"context"
	"time"

	"github.com/lthibault/log"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"go.uber.org/fx"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/casm/pkg/cluster/pulse"
	"github.com/wetware/casm/pkg/cluster/routing"
	"github.com/wetware/ww/pkg/csp"
	"github.com/wetware/ww/pkg/csp/proc"
)

type ClusterConfig struct {
	fx.In

	TTL          time.Duration        `optional:"true"`
	Meta         pulse.Preparer       `optional:"true"`
	RoutingTable cluster.RoutingTable `optional:"true"`
}

func (rc ClusterConfig) New(vat casm.Vat, r Router) (Cluster, error) {
	rt := rc.routingTable()

	err := r.RegisterTopicValidator(vat.NS, pulse.NewValidator(rt))
	if err != nil {
		return nil, err
	}

	t, err := r.Join(vat.NS)
	if err != nil {
		return nil, err
	}

	cr := &cluster.Router{
		Topic:        t,
		Log:          vat.Logger,
		TTL:          rc.TTL,
		Meta:         rc.Meta,
		RoutingTable: rt,
	}

	return router{
		Router: cr,
		r:      r,
	}, nil
}

func (rc ClusterConfig) routingTable() cluster.RoutingTable {
	if rc.RoutingTable == nil {
		rc.RoutingTable = routing.New(time.Now())
	}

	return rc.RoutingTable
}

type RuntimeConfig struct {
	fx.In

	Ctx    context.Context      `optional:"true"`
	Logger log.Logger           `optional:"true"`
	Config wazero.RuntimeConfig `optional:"true"`
}

func (rc RuntimeConfig) New() csp.Runtime {
	if rc.Ctx == nil {
		rc.Ctx = context.Background()
	}

	if rc.Config == nil {
		rc.Config = wazero.NewRuntimeConfig()
	}

	r := wazero.NewRuntimeWithConfig(rc.Ctx, rc.Config)
	wasi_snapshot_preview1.MustInstantiate(rc.Ctx, r)

	m := proc.BindModule(rc.Ctx, r,
		proc.WithLogger(rc.Logger),
		/* proc.WithClient(capnp.Client{}) */)

	return csp.Runtime{
		Runtime:    r,
		HostModule: m,
	}
}

// Router binds the lifecycle of CASM's *cluster.Router to that of the
// local Router interface.  This is needed because CASM requires us to
// seperately join the cluster topic and register a pulse.Validator on
// startup. In turn, this means we must *deregister* the validator and
// leave the cluster on shutdown.
type router struct {
	*cluster.Router
	r Router
}

func (r router) Close() error {
	r.Stop()
	return r.r.UnregisterTopicValidator(r.String())
}

func (r router) Bootstrap(ctx context.Context) error {
	return r.Router.Bootstrap(ctx)
}
