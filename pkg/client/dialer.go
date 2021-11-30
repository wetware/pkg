package client

import (
	"context"
	"fmt"
	"runtime"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wetware/casm/pkg/boot"
	"go.uber.org/fx"
)

type Dialer struct {
	fx.In

	Join discovery.Discoverer

	Host    HostFactory    `optional:"true"`
	Routing RoutingFactory `optional:"true"`
	PubSub  PubSubFactory  `optional:"true"`
	RPC     RPCFactory     `optional:"true"`
}

// Dial joins a cluster via 'addr', using the default Dialer.
func Dial(ctx context.Context, addr string) (*Node, error) {
	info, err := peer.AddrInfoFromString(addr)
	if err != nil {
		return nil, fmt.Errorf("addr: %w", err)
	}

	return DialDiscover(ctx, boot.StaticAddrs{*info})
}

// DialDiscover joins a cluster via the supplied discovery service,
// using the default dialer.
func DialDiscover(ctx context.Context, d discovery.Discoverer) (*Node, error) {
	return Dialer{Join: d}.Dial(ctx)
}

// Dial creates a client and connects it to a cluster.  The context
// can be safely cancelled when 'Dial' returns.
func (d Dialer) Dial(ctx context.Context) (*Node, error) {
	// Libp2p often binds the lifecycle of various types to that of
	// the context passed into their respective constructors (e.g.
	// host.Host).  Throughout the Wetware ecosystem, we enforce the
	// idiom that contexts passed to constructors are used to abort
	// construction, NOT shutdown the resulting type.
	//
	// This deferred call to 'cancel' is a guard against passing the
	// dial context to a constructor that expects long-lived context.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if d.Host == nil {
		d.Host = &BasicHostFactory{}
	}

	if d.Routing == nil {
		d.Routing = defaultRoutingFactory{}
	}

	if d.PubSub == nil {
		d.PubSub = defaultPubSubFactory{}
	}

	if d.RPC == nil {
		d.RPC = BasicRPCFactory{}
	}

	if rf, ok := d.Host.(RoutingHook); ok {
		d.Routing = &routingHook{RoutingFactory: d.Routing}
		rf.SetRouting(d.Routing)
	}

	var (
		n   = &Node{}
		err error
	)

	if n.h, err = d.Host.New(context.Background()); err != nil {
		return nil, fmt.Errorf("host: %w", err)
	}

	// Ensure we do not leak resources (e.g. bind addresses) if any
	// of our subsequent factories fail.
	//
	// Note that a 'HostFactory' implementation could potentially
	// return a 'Host' instsance that was previously created and is
	// still in use elsewhere. Under such conditions, closing would
	// be an error.
	//
	// Routing, PubSub and Conn are all tied to the lifetime of the
	// 'Host' instance.  They need not be explicitly closed.
	runtime.SetFinalizer(n, func(c *Node) { _ = c.h.Close() })

	if n.r, err = d.Routing.New(n.h); err != nil {
		return nil, fmt.Errorf("dht: %w", err)
	}

	if n.ps, err = d.PubSub.New(n.h, n.r); err != nil {
		return nil, fmt.Errorf("pubsub: %w", err)
	}

	if n.conn, err = d.RPC.New(ctx, n.h, d.Join); err != nil {
		return nil, fmt.Errorf("rpc: %w", err)
	}

	n.cap = n.conn.Bootstrap(ctx)
	return n, nil
}
