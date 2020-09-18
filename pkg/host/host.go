package host

import (
	"context"
	"time"

	"go.uber.org/fx"

	"github.com/libp2p/go-libp2p-core/event"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/spy16/parens"

	"github.com/wetware/ww/pkg/internal/filter"
	"github.com/wetware/ww/pkg/internal/rpc"
)

// Host .
type Host struct {
	ns   string
	r    routingTable
	host host.Host

	runtime interface {
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

	h.runtime = cfg.assemble(&h)
	return h, start(h.runtime)
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

// InterfaceListenAddrs returns a list of addresses at which this network
// listens. It expands "any interface" addresses (/ip4/0.0.0.0, /ip6/::) to
// use the known local interfaces.
func (h Host) InterfaceListenAddrs() ([]multiaddr.Multiaddr, error) {
	return h.host.Network().InterfaceListenAddresses()
}

// Peers in the cluster
func (h Host) Peers() peer.IDSlice {
	return append(h.r.Peers(), h.host.ID()) // TODO(xxx):  is the append necessary?
}

// Join the cluster via the specified peer information.
//
// Note that if the local peer belongs to a cluster, it will be merged with the remote
// peer's cluster.
func (h Host) Join(ctx context.Context, info peer.AddrInfo) error {
	return h.host.Connect(ctx, info)
}

// EventBus provides asynchronous notifications of changes in the host's internal state,
// or the state of the environment.
func (h Host) EventBus() event.Bus {
	return h.host.EventBus()
}

// Close the Host's network connections and stop its runtime processes.
// It is equivalent to calling Shutdown() with the default shutdown context, which
// expires in 15s.
func (h Host) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	return errors.Wrap(h.runtime.Stop(ctx), "host shutdown")
}

func start(r interface{ Start(context.Context) error }) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	return errors.Wrap(r.Start(ctx), "host start")
}

/*
	go.uber.org/fx
*/

type hostParams struct {
	fx.In

	Namespace string `name:"ns"`

	Host     host.Host
	Filter   filter.Filter
	Handlers []rpc.Capability `group:"rpc"`

	Env *parens.Env
}

func newHost(ctx context.Context, lx fx.Lifecycle, ps hostParams) Host {
	// export capabilities
	for _, cap := range ps.Handlers {
		ps.Host.SetStreamHandler(cap.Protocol(), handler(cap))
	}

	return Host{ns: ps.Namespace, host: ps.Host, r: ps.Filter}
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
