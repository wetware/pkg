package server

import (
	"context"
	"time"

	"github.com/lthibault/log"
	"go.uber.org/fx"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/casm/pkg/cluster/pulse"
	"github.com/wetware/casm/pkg/cluster/routing"
	"github.com/wetware/casm/pkg/debug"
)

type ClusterConfig struct {
	fx.In

	TTL          time.Duration        `optional:"true"`
	Meta         pulse.Preparer       `optional:"true"`
	RoutingTable cluster.RoutingTable `optional:"true"`
}

func (rc ClusterConfig) New(vat casm.Vat, r Router, log log.Logger) (Cluster, error) {
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
		Log:          log,
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

type DebugConfig struct {
	fx.In

	System   debug.SystemContext        `optional:"true" name:"debug-info"`
	Environ  func() []string            `optional:"true" name:"debug-environ"`
	Profiles map[debug.Profile]struct{} `optional:"true" name:"debug-profiles"`
}

func (dc DebugConfig) New() *debug.Server {
	return &debug.Server{
		Context:  dc.System,
		Environ:  dc.Environ,
		Profiles: dc.Profiles,
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
