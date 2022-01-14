package client

import (
	"context"
	"fmt"
	"path"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	"github.com/libp2p/go-libp2p/config"
	"github.com/lthibault/log"

	"github.com/wetware/casm/pkg/boot"
	ww "github.com/wetware/ww/pkg"
)

type RoutingFactory func(host.Host) (routing.Routing, error)

type Dialer struct {
	ns       string
	log      log.Logger
	hostOpts []config.Option
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

	n.ns = d.ns

	n.h, err = libp2p.New(context.Background(), d.hostOpts...)
	if err != nil {
		return
	}

	if n.ps.Client, err = d.newRootCapability(ctx, n.h, join); err == nil {
		// Block until the client is fully resolved. This is necessary because
		// returning from 'Dial' will cause 'ctx' to be canceled, aborting the
		// bootstrap process.
		err = n.ps.Client.Resolve(ctx)
	}

	return
}

func (d Dialer) newRootCapability(ctx context.Context, h host.Host, join discovery.Discoverer) (*capnp.Client, error) {
	peers, err := join.FindPeers(ctx, d.ns)
	if err != nil {
		return nil, err
	}

	var s network.Stream
	for info := range peers {
		if err = h.Connect(ctx, info); err != nil {
			continue
		}

		s, err = h.NewStream(ctx, info.ID,
			ww.Subprotocol(d.ns, "packed"),
			ww.Subprotocol(d.ns))
		if err == nil {
			break
		}
	}

	if err != nil {
		defer h.Close()
		return nil, err
	}

	conn := rpc.NewConn(transportFor(s), &rpc.Options{
		// // TODO: authenticate the base capability set for this client.
		// //       One possible approach is to return a client that implements
		// //       zero or more <Cap>Auth methods, e.g. PubSubAuth.  The
		// //       receiver would then attempt to call all <Cap>Auth methods it
		// //       knows about, and would receive a "not implemented" exception
		// //       if the client is not requesting the corresponding capability.
		// //
		// //       Importantly, when the client _is_ requesting a capability,
		// //       the corresponding <Cap>Auth method would have to return a
		// //       capability that is somehow able to prove that the sender is
		// //       authorized to obtain the requested capability.  We consider
		// //       two possible approaches:
		// //
		// //       1)  A SturdyRef behaving like a secret API key.  The advantage
		// //           of this method is that it is simple.  Because the SturdyRef
		// //           is secret, however, it ought not be transmitted in plain
		// //           text.  One solution is to transmit a salted hash of the API
		// //           key, which somewhat complexifies implementation and has
		// //           performance implications for lookup performance.
		// //
		// //           The main drawback to this approach is that it requires
		// //           each cluster node to maintain a stateful, dynamic — and
		// //           more importantly, *consistent* — database of valid refs.
		// //           This goes against the general ethos of Wetware as a PA/EL
		// //           system, and introduces a nontrivial attack vector wherein
		// //           an adversary could obtain capabilities via stale keys,
		// //           e.g. by conducting an eclipse attack that prevents the
		// //           targeted node from updating its SturdyRef table.
		// //           Moreover, the dynamic and rapidly-chaning nature of the
		// //           SturdyRef table preculdes the use of blockchain-based
		// //           state management.
		// //
		// //       2)  A capability that takes a cryptographically-random nonce
		// //           as input and blindly (!) returns a record.Record instance
		// //           instance that contains the nonce and is that is signed by
		// //           a trusted authority. The domain string of the record binds
		// //           the authority to a specific capability.  For example, an
		// //           attempt to use a valid, signed record to authenticate the
		// //           PubSub capability would fail if the record's somain did
		// //           not match "ww.cap.pubsub".
		// //
		// //           The advantage to this approach is that state-management
		// //           is reduced to a small set of long-lived key pairs, which
		// //           can even be configured statically for small clusters.
		// //           Where static configuration is not desireable, the small
		// //           size of data and infrequent updates make this solution
		// //           amenable to a variety of blockchain-based approaches.
		// BootstrapClient: authProvider.Client,
	})

	return conn.Bootstrap(ctx), nil
}

func transportFor(s network.Stream) rpc.Transport {
	if path.Base(string(s.Protocol())) == "packed" {
		return rpc.NewPackedStreamTransport(s)
	}

	return rpc.NewStreamTransport(s)
}

type addrString string

func (addr addrString) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	info, err := peer.AddrInfoFromString(string(addr))
	if err != nil {
		return nil, fmt.Errorf("addr: %w", err)
	}

	return boot.StaticAddrs{*info}.FindPeers(ctx, ns, opt...)
}
