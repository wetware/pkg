package boot

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/peer"
)

type Cache struct {
	Match func(string) bool
	Cache discovery.Discovery
	Else  discovery.Discovery
}

func (c Cache) Advertise(ctx context.Context, ns string, opt ...discovery.Option) (time.Duration, error) {
	if c.Match(ns) {
		return c.Cache.Advertise(ctx, ns, opt...)
	}

	return c.Else.Advertise(ctx, ns, opt...)
}

func (c Cache) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	if c.Match(ns) {
		return c.Cache.FindPeers(ctx, ns, opt...)
	}

	return c.Else.FindPeers(ctx, ns, opt...)
}

// import (
// 	"context"
// 	"fmt"
// 	"math/rand"
// 	"time"

// 	"github.com/libp2p/go-libp2p-core/discovery"
// 	"github.com/libp2p/go-libp2p-core/peer"
// 	syncutil "github.com/lthibault/util/sync"
// )

// // ViewProvider can provide a snapshot of an overlay's view.
// type ViewProvider interface {
// 	View() ([]peer.AddrInfo, error)
// }

// // PassiveOverlay provides a local view of the cluster.  It is
// // said to be 'passive' since implementations need not directly
// // track the liveness of hosts. View MUST remain valid when the
// // network is down.
// type PassiveOverlay interface {
// 	// String returns the namespace associated with the provider.
// 	String() string

// 	// Join adds the peer to the cache.
// 	Join(context.Context, peer.AddrInfo) error

// 	ViewProvider
// }

// // Dual intercepts calls to 'Advertise' and 'FindPeers' and dynamically
// // dispatches them to the 'Boot' if the namespace matches the output of
// // its String() method.  Else, the call is dispatched to the fallback.
// //
// // A common use case is for cluster bootstrap, where the cluster topic
// // should draw from a cache (and fall back on a bootstrap service such
// // as MDNS), but all other topics should query a DHT.
// type Dual struct {
// 	Boot interface {
// 		String() string
// 		discovery.Discovery
// 	}

// 	discovery.Discovery
// }

// // String returns the namespace associated with the Cache.
// // Calls to FindPeers with the parameter ns = c.String() will be cached.
// // All others will be passed to D.
// func (d Dual) String() string { return d.Boot.String() }

// // Advertise using the boot cache if ns == d.String(), else using D.
// func (d Dual) Advertise(ctx context.Context, ns string, opt ...discovery.Option) (time.Duration, error) {
// 	return d.discovery(ns).Advertise(ctx, ns, opt...)
// }

// // FindPeers using the boot cache if ns == d.String(), else using D.
// func (d Dual) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
// 	return d.discovery(ns).FindPeers(ctx, ns, opt...)
// }

// func (d Dual) discovery(ns string) discovery.Discovery {
// 	if normalizeNS(ns) == d.String() {
// 		return d.Boot
// 	}

// 	return d.Discovery
// }

// // Cache wraps a discovery.Discovery and a RecordProvider to cache peer
// // addresses locally. It can be used to repair partitions or rejoin the
// // cluster after having been orphaned.
// //
// // Cache intercepts calls to FindPeers and dispatches them to the cache
// // iff ns == Cache.String(). Calls to Advertise are passed directly the
// // embedded Discovery service.
// type Cache struct {
// 	discovery.Discovery
// 	Cache PassiveOverlay

// 	// OnErr is called when a cache update is triggered, but none of the
// 	// discovered peers could be joined.  This is usually indicative of
// 	// a network problem.
// 	//
// 	// Note that OnErr WILL NOT be called if no peers were found.
// 	OnErr func(error)
// }

// // String returns the namespace associated with the Cache.
// // Calls to FindPeers with the parameter ns = c.String() will be cached.
// // All others will be passed to D.
// func (c Cache) String() string { return c.Cache.String() }

// // FindPeers providing the service 'ns'.  If ns == Cache.String(), the
// // service attempts to return values from cache. If none are found, it
// // falls back on D, and uses any peers found to populate the cache.
// //
// // Callers MUST ensure that 'ctx' eventually expires, or FindPeers may
// // leak goroutines.
// func (c Cache) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
// 	os := &discovery.Options{}
// 	if err := os.Apply(opt...); err != nil {
// 		return nil, err
// 	}

// 	// try cache before falling back on d?
// 	if c.isClusterNS(ns) {
// 		if ps, err := c.lookup(); len(ps) > 0 || err != nil {
// 			return staticChan(limited(os, ps)), err
// 		}
// 	}

// 	ch, err := c.Discovery.FindPeers(ctx, ns, opt...)
// 	if err == nil && c.isClusterNS(ns) {
// 		ch = c.interceptAndJoin(ctx, ch)
// 	}

// 	return ch, err
// }

// func (c Cache) isClusterNS(ns string) bool { return c.String() == ns }

// func (c Cache) lookup() ([]peer.AddrInfo, error) {
// 	view, err := c.Cache.View()

// 	rand.Shuffle(len(view), func(i, j int) {
// 		view[i], view[j] = view[j], view[i]
// 	})

// 	return view, err
// }

// func (c Cache) interceptAndJoin(ctx context.Context, ch <-chan peer.AddrInfo) <-chan peer.AddrInfo {
// 	out := make(chan peer.AddrInfo, cap(ch))
// 	lim := make(chan struct{}, 8) // rate-limiter

// 	go func() {
// 		defer close(lim)
// 		defer close(out)

// 		var any syncutil.Any
// 		for info := range ch {
// 			select {
// 			case <-lim:
// 				any.Go(c.join(ctx, lim, out, info))
// 			case <-ctx.Done():
// 				// We assume that 'ctx' is tied to the lifetime of 'ch', and that
// 				// the next iteration will break out of the loop.
// 			}

// 		}

// 		if err := any.Wait(); err != nil {
// 			c.onErr(fmt.Errorf("failed to populate cache: %w", err))
// 		}
// 	}()

// 	return out
// }

// func (c Cache) join(ctx context.Context, lim chan<- struct{}, out chan<- peer.AddrInfo, info peer.AddrInfo) func() error {
// 	return func() (err error) {
// 		defer func() {
// 			select {
// 			case lim <- struct{}{}:
// 			default:
// 			}
// 		}()

// 		if err = c.Cache.Join(ctx, info); err == nil {
// 			select {
// 			case out <- info:
// 			case <-ctx.Done():
// 				err = ctx.Err()
// 			}
// 		}

// 		return
// 	}
// }

// func (c Cache) onErr(err error) {
// 	if c.OnErr != nil {
// 		c.OnErr(err)
// 	}
// }
