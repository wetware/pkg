package client

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	ps "github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/pnet"
	"github.com/libp2p/go-libp2p-core/routing"
	disc "github.com/libp2p/go-libp2p-discovery"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/lthibault/log"
	ctxutil "github.com/lthibault/util/ctx"
	"github.com/wetware/casm/pkg/boot"
)

type RoutingFactory func(host.Host) (routing.Routing, error)

type Dialer struct {
	ns  string
	log log.Logger

	host   host.Host
	secret pnet.PSK
	auth   connmgr.ConnectionGater

	pubsub PubSub

	newRouting RoutingFactory
}

func NewDialer(opt ...Option) Dialer {
	var d Dialer
	for _, option := range withDefault(opt) {
		option(&d)
	}
	return d
}

// Dial joins a cluster via 'addr', using the default Dialer.
func Dial(ctx context.Context, addr string, opt ...Option) (Node, error) {
	return DialDiscover(ctx, addrString(addr), opt...)
}

// DialDiscover joins a cluster via the supplied discovery service,
// using the default dialer.
func DialDiscover(ctx context.Context, d discovery.Discoverer, opt ...Option) (Node, error) {
	return NewDialer(opt...).Dial(ctx, d)
}

// Dial creates a client and connects it to a cluster.  The context
// can be safely cancelled when 'Dial' returns.
func (d Dialer) Dial(ctx context.Context, join discovery.Discoverer) (n Node, err error) {
	// Enforce the semantic convention that 'ctx' is valid only for the duration
	// of the call to 'Dial'.  Processes that outlive this call should use their
	// own contexts.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if n.host = d.host; n.host == nil {
		if n.host, err = d.newHost(&n); err != nil {
			return
		}

		// If the user explicitly passed in a host using the 'WithHost'
		// option, they are responsible for closing it.
		defer func() {
			if err != nil {
				n.host.Close()
			}
		}()
	}

	if n.routing, err = d.newRouting(n.host); err != nil {
		return
	}

	n.host = routedhost.Wrap(n.host, n.routing)

	func() {
		// Calls to the discovery service MUST block until 'overlay' has
		// been assigned to 'n'.  'Node.bootstrapRequired' will block
		// on 'ctx.Done()' until this function returns.
		//
		// Note that the anonymous function is necessary to avoid a
		// deadlock in 'n.bootstrap'.
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		if n.overlay, err = d.newOverlay(ctx, &n, join); err != nil {
			return
		}
	}()

	if err = n.bootstrap(ctx, n.host, join); err != nil {
		return
	}

	return
}

func (d Dialer) newHost(n *Node) (host.Host, error) {
	var opt = []libp2p.Option{
		libp2p.NoListenAddrs,
		libp2p.NoTransports,
		libp2p.Transport(libp2pquic.NewTransport)}

	if d.secret != nil {
		opt = append(opt, libp2p.PrivateNetwork(d.secret))
	}

	if d.auth != nil {
		opt = append(opt, libp2p.ConnectionGater(d.auth))
	}

	return libp2p.New(context.Background(), opt...)
}

func (d Dialer) newOverlay(ctx context.Context, n *Node, join discovery.Discoverer) (ov overlay, err error) {
	if ov.PubSub = d.pubsub; ov.PubSub == nil {
		if ov.PubSub, err = d.newPubSub(ctx, n, join); err != nil {
			return
		}
	}

	if ov.t, err = ov.PubSub.Join(d.ns); err != nil {
		return
	}

	if ov.stop, err = ov.t.Relay(); err != nil {
		return
	}

	return
}

func (d Dialer) newPubSub(ctx context.Context, n *Node, join discovery.Discoverer) (*pubsub.PubSub, error) {
	return pubsub.NewGossipSub(ctxFromHost(n.host), n.host,
		pubsub.WithDiscovery(d.newDiscovery(ctx, n, join)))
}

func ctxFromHost(h host.Host) context.Context {
	return ctxutil.C(h.Network().Process().Closing())
}

// newDiscovery returns wraps 'join' and returns a discovery service suitable
// for use in 'PubSub'.  It routes calls to 'Advertise' and 'Discover' so as
// to rely on 'join' only during bootstrap, otherwise preferring the DHT.
func (d Dialer) newDiscovery(ctx context.Context, n *Node, join discovery.Discoverer) discovery.Discovery {
	// Bootstrapper is used for discovery operations on the 'd.ns' namespace
	// when the pubsub service is disconnected from the cluster topic.
	//
	// Note that because client hosts do not listen for incoming connections,
	// calls to 'Advertise' MUST be a noop, otherwise server and client nodes
	// alike may try to connect to the client.
	bootstrapper := struct {
		discovery.Discoverer
		discovery.Advertiser
	}{
		Discoverer: join,
		Advertiser: nopAdvertiser{},
	}

	// Route calls to the bootstrapper when they involve the cluster namespace
	// and the pubsub overlay has no peers in the corresponding topic.
	return boot.Namespace{
		Match:   n.bootstrapRequired(ctx),
		Target:  bootstrapper,
		Default: disc.NewRoutingDiscovery(n.routing),
	}
}

// returns a matcher that returns 'true' if the namespace
// matches the cluster topic and the pubsub is currently
// disconnected from the overlay.
func (n *Node) bootstrapRequired(ctx context.Context) func(string) bool {
	// overlay's fields are nil when 'bootstrapRequried' is called, so
	// we block all calls to the matcher function until 'Dial' has
	// returned.
	var ready = ctx.Done()

	return func(s string) bool {
		<-ready // closed by 'Dial'
		return n.overlay.DiscoveryString() == s && n.overlay.Orphaned()
	}
}

type overlay struct {
	PubSub
	stop pubsub.RelayCancelFunc
	t    *pubsub.Topic
}

func (o overlay) Close() error {
	o.stop() // MUST happen before o.t.Close()
	return o.t.Close()
}

// String returns the cluster namespace
func (o overlay) String() string {
	return o.t.String()
}

func (o overlay) Orphaned() bool {
	return len(o.ListPeers(o.t.String())) == 0
}

func (o overlay) DiscoveryString() string {
	return "floodsub:" + o.t.String()
}

func (n *Node) bootstrap(ctx context.Context, h host.Host, join discovery.Discoverer) error {
	peers, err := join.FindPeers(ctx, n.String())
	if err != nil {
		return err
	}

	for info := range peers {
		if err = h.Connect(ctx, info); err == nil {
			break
		}
	}

	if err != nil {
		return err
	}

	return n.routing.Bootstrap(ctx)
}

type addrString string

func (addr addrString) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	info, err := peer.AddrInfoFromString(string(addr))
	if err != nil {
		return nil, fmt.Errorf("addr: %w", err)
	}

	return boot.StaticAddrs{*info}.FindPeers(ctx, ns, opt...)
}

type nopAdvertiser struct{}

func (nopAdvertiser) Advertise(context.Context, string, ...discovery.Option) (time.Duration, error) {
	return ps.PermanentAddrTTL, nil
}
