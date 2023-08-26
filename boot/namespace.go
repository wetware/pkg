package boot

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/exp/slog"
)

type Namespace struct {
	Name               string
	Bootstrap, Ambient discovery.Discovery
}

func (n Namespace) Network() string {
	return "floodsub:" + n.Name
}

func (n Namespace) Advertise(ctx context.Context, ns string, opt ...discovery.Option) (time.Duration, error) {
	logger := slog.Default().With(
		"ns", n.Name,
		"net", ns)

	ttl, err := n.Ambient.Advertise(ctx, ns, opt...)
	if err != nil {
		logger.With("error", err)
	}
	defer logger.Debug("advertised")

	return ttl, err
}

func (n Namespace) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	logger := slog.Default().With(
		"ns", n.Name,
		"match", ns)

	peers, err := n.Ambient.FindPeers(ctx, ns, opt...)
	if err != nil {
		logger.With("error", err)
	}
	defer logger.Debug("crawled")

	return peers, err
}
