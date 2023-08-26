package boot

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Namespace struct {
	Name      string
	Bootstrap discovery.Discovery
	Ambient   discovery.Discovery
}

func (n Namespace) Network() string {
	return n.Name
}

func (n Namespace) Advertise(ctx context.Context, ns string, opt ...discovery.Option) (time.Duration, error) {
	if n.Match(ns) {
		return n.Bootstrap.Advertise(ctx, ns, opt...)
	}

	return n.Ambient.Advertise(ctx, ns, opt...)
}

func (n Namespace) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	if n.Match(ns) {
		return n.Bootstrap.FindPeers(ctx, ns, opt...)
	}

	return n.Ambient.FindPeers(ctx, ns, opt...)
}

func (n Namespace) Match(ns string) bool {
	return ns == "floodsub:"+n.Name
}
