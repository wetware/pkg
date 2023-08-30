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

func (n Namespace) Advertise(ctx context.Context, ns string, opt ...discovery.Option) (ttl time.Duration, err error) {
	slog.Debug("advertising",
		"name", n.Name,
		"ns", ns)

	if strings.TrimPrefix(ns, "floodsub:") == n.Name {
		return n.Bootstrap.Advertise(ctx, n.Name, opt...)
	}

	return n.Ambient.Advertise(ctx, n.Name, opt...)
}

func (n Namespace) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (out <-chan peer.AddrInfo, err error) {
	slog.Debug("finding peers",
		"name", n.Name,
		"ns", ns)

	if strings.TrimPrefix(ns, "floodsub:") == n.Name {
		return n.Bootstrap.FindPeers(ctx, n.Name, opt...)
	}

	return n.Ambient.FindPeers(ctx, n.Name, opt...)
}
