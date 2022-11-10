package server

import (
	"time"

	"github.com/lthibault/log"
	"go.uber.org/fx"

	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/casm/pkg/cluster/pulse"
	"github.com/wetware/casm/pkg/cluster/routing"
	"github.com/wetware/casm/pkg/debug"
)

type RoutingConfig struct {
	fx.In

	Log          log.Logger           `optional:"true"`
	TTL          time.Duration        `optional:"true"`
	Meta         pulse.Preparer       `optional:"true"`
	RoutingTable cluster.RoutingTable `optional:"true"`
}

func (rc RoutingConfig) Bind(r Router, ns string) (*cluster.Router, error) {
	rt := rc.routingTable()

	err := r.RegisterTopicValidator(ns, pulse.NewValidator(rt))
	if err != nil {
		return nil, err
	}

	t, err := r.Join(ns)
	return &cluster.Router{
		Topic:        t,
		Log:          rc.Log,
		TTL:          rc.TTL,
		Meta:         rc.Meta,
		RoutingTable: rt,
	}, err
}

func (rc RoutingConfig) routingTable() cluster.RoutingTable {
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
