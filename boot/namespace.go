package boot

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Namespace struct {
	Match   func(string) bool
	Target  discovery.Discovery
	Default discovery.Discovery
}

func (n Namespace) Advertise(ctx context.Context, ns string, opt ...discovery.Option) (time.Duration, error) {
	if n.Match(ns) {
		return n.Target.Advertise(ctx, ns, opt...)
	}

	return n.Default.Advertise(ctx, ns, opt...)
}

func (n Namespace) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	if n.Match(ns) {
		return n.Target.FindPeers(ctx, ns, opt...)
	}

	return n.Default.FindPeers(ctx, ns, opt...)
}
