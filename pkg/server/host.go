package server

import (
	"context"
	"time"

	"go.uber.org/fx"

	"github.com/libp2p/go-libp2p-core/event"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/lthibault/wetware/pkg/internal/routing"
	"github.com/lthibault/wetware/pkg/internal/rpc"
)

// Host .
type Host struct {
	ns   string
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

// Loggable fields for the Host
func (h Host) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"ns":    h.ns,
		"id":    h.ID(),
		"addrs": h.Addrs(),
	}
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	return h.app.Start(ctx)
}

// Close the Host's network connections and stop its runtime processes.
func (h Host) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	return h.app.Stop(ctx)
}

/*
	go.uber.org/fx
*/

type hostParams struct {
	fx.In

	Namespace string `name:"ns"`

	Host    host.Host
	Routing routing.Table
}

func newHost(ctx context.Context, lx fx.Lifecycle, ps hostParams) Host {
	// export capabilities
	for _, cap := range []rpc.Capability{
		newRootAnchor(ps.Host, ps.Routing),
	} {
		ps.Host.SetStreamHandler(cap.Protocol(), handler(cap))
	}

	return Host{ns: ps.Namespace, host: ps.Host, r: ps.Routing}
}

func handler(cap rpc.Capability) network.StreamHandler {
	return func(s network.Stream) {
		defer s.Reset()

		if err := rpc.Handle(cap, s); err != nil {
			panic(err) // TODO(easy): emit to event bus
			// log.WithFields(cap.Loggable()).
			// 	WithError(err).
			// 	Debug("connection aborted")
		}
	}
}
