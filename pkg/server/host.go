package server

import (
	"context"

	log "github.com/lthibault/log/pkg"
	"go.uber.org/fx"

	"github.com/libp2p/go-libp2p-core/event"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/lthibault/wetware/pkg/internal/routing"
	"github.com/lthibault/wetware/pkg/internal/rpc"
)

// Host .
type Host struct {
	log  log.Logger
	r    routing.Table
	host host.Host

	app interface {
		Start(context.Context) error
		Stop(context.Context) error
	}
}

// New Host
func New(opt ...Option) (h Host, err error) {
	var cfg Config
	for _, f := range withDefault(opt) {
		if err = f(&cfg); err != nil {
			return
		}
	}

	cfg.assemble(&h)
	return
}

// Log returns the host's logger
func (h Host) Log() log.Logger {
	return h.log.WithFields(log.F{
		"id":    h.ID(),
		"addrs": h.Addrs(),
	})
}

// ID of the Host
func (h Host) ID() peer.ID {
	return h.host.ID()
}

// Addrs on which the host is reachable
func (h Host) Addrs() []multiaddr.Multiaddr {
	return h.host.Addrs()
}

// Peers in the cluster
func (h Host) Peers() peer.IDSlice {
	return append(h.r.Peers(), h.host.ID())
}

// EventBus provides asynchronous notifications of changes in the host's internal state,
// or the state of the environment.
func (h Host) EventBus() event.Bus {
	return h.host.EventBus()
}

// Start the Host's network connections and start its runtime processes.
func (h Host) Start() error {
	return h.app.Start(context.Background())
}

// Close the Host's network connections and stop its runtime processes.
func (h Host) Close() error {
	return h.app.Stop(context.Background())
}

/*
	go.uber.org/fx
*/

type hostParams struct {
	fx.In

	Log log.Logger

	Host    host.Host
	Routing routing.Table
}

func newHost(ctx context.Context, lx fx.Lifecycle, ps hostParams) Host {
	// export capabilities
	for _, cap := range []rpc.Capability{
		newRootAnchor(ps.Log, ps.Host, ps.Routing),
	} {
		ps.Host.SetStreamHandler(cap.Protocol(), rpc.Export(cap))
	}

	return Host{log: ps.Log, host: ps.Host, r: ps.Routing}
}
