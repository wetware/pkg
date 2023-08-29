package boot

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/wetware/pkg/util/proto"
)

type Namespace struct {
	Name               string
	Bootstrap, Ambient discovery.Discovery
}

func (n Namespace) Protocols() []protocol.ID {
	return proto.Namespace(n.Name)
}

func (n Namespace) Advertise(ctx context.Context, net string, opt ...discovery.Option) (ttl time.Duration, err error) {
	slog.Debug("advertising",
		"ns", n.Name,
		"net", net)

	if strings.HasPrefix(net, "floodsub:") {
		return trimPrefix{n.Bootstrap}.Advertise(ctx, net, opt...)
	}

	return n.Ambient.Advertise(ctx, n.Name, opt...)
}

func (n Namespace) FindPeers(ctx context.Context, net string, opt ...discovery.Option) (out <-chan peer.AddrInfo, err error) {
	slog.Debug("finding peers",
		"ns", n.Name,
		"net", net)

	if strings.HasPrefix(net, "floodsub:") {
		return trimPrefix{n.Bootstrap}.FindPeers(ctx, net, opt...)
	}

	return n.Ambient.FindPeers(ctx, n.Name, opt...)
}

// Trims the "floodsub:" prefix from the namespace.  This is needed because
// clients do not use pubsub, and will search for the exact namespace string.
type trimPrefix struct {
	discovery.Discovery
}

func (b trimPrefix) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	ns = strings.TrimPrefix(ns, "floodsub:")
	return b.Discovery.FindPeers(ctx, ns, opt...)
}

func (b trimPrefix) Advertise(ctx context.Context, ns string, opt ...discovery.Option) (time.Duration, error) {
	ns = strings.TrimPrefix(ns, "floodsub:")
	return b.Discovery.Advertise(ctx, ns, opt...)
}
