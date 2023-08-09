package boot

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/peer"

	ps "github.com/libp2p/go-libp2p/core/peerstore"

	ma "github.com/multiformats/go-multiaddr"
)

var (
	_ discovery.Discoverer = StaticAddrs(nil)
)

// StaticAddrs is a set of bootstrap-peer addresses.
type StaticAddrs []peer.AddrInfo

// NewStaticAddrs from one or more multiaddrs.
func NewStaticAddrs(as ...ma.Multiaddr) (StaticAddrs, error) {
	return peer.AddrInfosFromP2pAddrs(as...)
}

func NewStaticAddrStrings(ss ...string) (as StaticAddrs, _ error) {
	for _, s := range ss {
		info, err := peer.AddrInfoFromString(s)
		if err != nil {
			return nil, err
		}

		as = append(as, *info)
	}

	return as, nil
}

func (as StaticAddrs) Len() int           { return len(as) }
func (as StaticAddrs) Less(i, j int) bool { return as[i].ID < as[j].ID }
func (as StaticAddrs) Swap(i, j int)      { as[i], as[j] = as[j], as[i] }

func (as StaticAddrs) Filter(f func(peer.AddrInfo) bool) StaticAddrs {
	filt := make(StaticAddrs, 0, len(as))
	for _, info := range as {
		if f(info) {
			filt = append(filt, info)
		}
	}
	return filt
}

// Advertise is a nop that defaults to PermanentAddrTTL.
func (as StaticAddrs) Advertise(_ context.Context, _ string, opt ...discovery.Option) (time.Duration, error) {
	opts := &discovery.Options{}
	if err := opts.Apply(opt...); err != nil {
		return 0, err
	}
	if opts.Ttl == 0 {
		opts.Ttl = ps.PermanentAddrTTL
	}
	return opts.Ttl, nil
}

// FindPeers converts the static addresses into AddrInfos
func (as StaticAddrs) FindPeers(_ context.Context, _ string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	opts := &discovery.Options{}
	if err := opts.Apply(opt...); err != nil {
		return nil, err
	}

	return staticChan(limited(opts, as)), nil
}

// Close is a nop method.  It exists to satisfy boot.DiscoveryCloser.
func (StaticAddrs) Close() error { return nil }

func limited(opts *discovery.Options, ps []peer.AddrInfo) []peer.AddrInfo {
	if opts.Limit > 0 && opts.Limit < len(ps) {
		ps = ps[:opts.Limit]
	}
	return ps
}

func staticChan(ps []peer.AddrInfo) chan peer.AddrInfo {
	ch := make(chan peer.AddrInfo, len(ps))
	for _, info := range ps {
		ch <- info
	}
	close(ch)
	return ch
}
