package server

import (
	"context"
	"sync"
	"time"

	log "github.com/lthibault/log/pkg"
	syncutil "github.com/lthibault/util/sync"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"

	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/event"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	discovery "github.com/libp2p/go-libp2p-discovery"
	discover "github.com/lthibault/wetware/pkg/discover"

	ww "github.com/lthibault/wetware/pkg"
)

/*
	connpolicy.go contains the logic responsible for ensuring cluster connectivity.  It
	it enacts a policy that attempts to maintain between kmin and kmax unique
	connections.
*/

const (
	tagProtectKmin = "ww-protect-kmin"
	tagStreamInUse = "ww-stream-in-use"
)

type connpolicyConfig struct {
	fx.In

	Ctx context.Context
	Log log.Logger

	Host    host.Host
	ConnMgr connmgr.ConnManager

	Namespace string `name:"ns"`
	KMin      int    `name:"kmin"`
	KMax      int    `name:"kmax"`

	Boot      discover.Protocol
	Discovery discovery.Discovery
}

// connpolicy maintains a bounded set of connections to peers, ensuring cluster
// connectivity.
func connpolicy(lx fx.Lifecycle, cfg connpolicyConfig) error {
	bus := cfg.Host.EventBus()

	if err := protect(lx, cfg, bus); err != nil {
		return err
	}

	return maintain(lx, cfg, bus)
}

func protect(lx fx.Lifecycle, cfg connpolicyConfig, bus event.Bus) error {
	sub, err := bus.Subscribe([]interface{}{
		new(ww.EvtNeighborhoodChanged),
		new(ww.EvtStreamChanged),
	})
	if err != nil {
		return err
	}
	lx.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return sub.Close()
		},
	})

	policy := connProtectionPolicy{
		kmin: cfg.KMin,
		cm:   cfg.ConnMgr,
		sub:  sub,
	}
	go policy.loop()
	return nil
}

type connProtectionPolicy struct {
	kmin int
	cm   connmgr.ConnManager
	sub  event.Subscription
}

func (p connProtectionPolicy) loop() {
	for v := range p.sub.Out() {
		switch ev := v.(type) {
		case ww.EvtNeighborhoodChanged:
			// policy is to protect a connection if it's one of the first kmin.
			p.setProtectStatus(ev)
		case ww.EvtStreamChanged:
			// policy is to prune connections with the fewest open streams.
			p.setTag(ev)
		}

	}
}

func (p connProtectionPolicy) setProtectStatus(ev ww.EvtNeighborhoodChanged) {
	switch ev.State {
	case ww.ConnStateOpened:
		if ev.From == ww.PhasePartial {
			p.cm.Protect(ev.Peer, tagProtectKmin)
		}
	case ww.ConnStateClosed:
		p.cm.Unprotect(ev.Peer, tagProtectKmin)
		p.cm.UntagPeer(ev.Peer, tagStreamInUse)
	}
}

func (p connProtectionPolicy) setTag(ev ww.EvtStreamChanged) {
	switch ev.State {
	case ww.StreamStateOpened:
		p.cm.TagPeer(ev.Peer, tagStreamInUse, 1)
	case ww.StreamStateClosed:
		p.cm.TagPeer(ev.Peer, tagStreamInUse, -1)
	}
}

func maintain(lx fx.Lifecycle, cfg connpolicyConfig, bus event.Bus) error {
	sub, err := bus.Subscribe(new(ww.EvtNeighborhoodChanged))
	if err != nil {
		return err
	}

	lx.Append(fx.Hook{
		OnStart: func(context.Context) error {
			m := neighborhoodMaintainer{
				log:  cfg.Log,
				ns:   cfg.Namespace,
				kmin: cfg.KMin,
				kmax: cfg.KMax,
				host: cfg.Host,
				b:    cfg.Boot,
				d:    cfg.Discovery,
			}
			go m.loop(cfg.Ctx, sub)
			return nil
		},
		OnStop: func(context.Context) error {
			return sub.Close()
		},
	})

	return nil
}

type neighborhoodMaintainer struct {
	log log.Logger

	ns         string
	kmin, kmax int

	host host.Host

	sf singleflight
	b  discover.Strategy
	d  discovery.Discoverer
}

func (m *neighborhoodMaintainer) loop(ctx context.Context, sub event.Subscription) {
	ticker := time.NewTicker(time.Second * 15)
	defer ticker.Stop()

	var (
		ev     ww.EvtNeighborhoodChanged
		reqctx context.Context
		cancel context.CancelFunc
	)

	for {
		switch ev.To {
		case ww.PhaseOrphaned:
			reqctx, cancel = context.WithCancel(ctx)
			m.join(reqctx)
		case ww.PhasePartial:
			reqctx, cancel = context.WithCancel(ctx)
			m.graft(reqctx, m.kmin-ev.N)
		case ww.PhaseOverloaded:
			// In-flight requests only become a liability when the host is overloaded.
			//
			// - Partially-connected nodes still benefit from in-flight join requests.
			// - Recently-orphaned nodes still benefit from in-flight graft requests.
			// - In-flight requests are harmless to completely-connected nodes; excess
			//   connections will be pruned by the connection manager, at worst.
			cancel()
		}

		select {
		case <-ticker.C:
		case v, ok := <-sub.Out():
			if ok {
				ev = v.(ww.EvtNeighborhoodChanged)
			}
		case <-ctx.Done():
			cancel()
			return
		}
	}
}

func (m *neighborhoodMaintainer) join(ctx context.Context) {
	go m.sf.Do("join", func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second*30)
		defer cancel()
		defer m.sf.Reset("join")

		ps, err := m.b.DiscoverPeers(ctx,
			discover.WithLogger(m.log),
			discover.WithLimit(3))
		if err != nil {
			m.log.WithError(err).Debug("peer discovery failed")
			return
		}

		var any syncutil.Any
		for info := range ps {
			if info.ID == m.host.ID() {
				continue // got our own addr info; skip.
			}

			any.Go(m.connect(ctx, info))
		}

		if err = any.Wait(); err == nil {
			err = ctx.Err() // Wait might return nil if no peers were found.
		}

		if err != nil {
			m.log.WithError(err).Debug("join failed")
		}
	})
}

// TODO:  this needs work.  seems like parts of this are broken ...
func (m *neighborhoodMaintainer) graft(ctx context.Context, limit int) {
	go m.sf.Do("graft", func() {
		discoverCtx, cancel := context.WithTimeout(ctx, time.Second*30)
		defer cancel()
		defer m.sf.Reset("graft")

		// TODO:  provide value for `m.ns` in the DHT.
		ch, err := m.d.FindPeers(discoverCtx, m.ns, discovery.Limit(limit))
		if err != nil {
			m.log.WithError(err).Debug("discovery failed")
			return
		}

		var g errgroup.Group
		for info := range ch {
			// TODO:  filter out self, and filter out already-connected peers.

			g.Go(m.connect(ctx, info))
		}

		if err = g.Wait(); err != nil {
			m.log.WithError(err).Debug("graft failed")
		}
	})
}

func (m *neighborhoodMaintainer) connect(ctx context.Context, info peer.AddrInfo) func() error {
	return func() error {
		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		if err := m.host.Connect(ctx, info); err != nil {
			m.log.WithError(err).
				WithField("peer", info.ID).
				Trace("connection attempt failed")
			return err
		}

		m.log.WithField("peer", info.ID).
			Trace("connection established")
		return nil
	}
}

type singleflight struct {
	mu sync.Mutex
	m  map[string]*sync.Once
}

func (sf *singleflight) Do(key string, f func()) {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if sf.m == nil {
		sf.m = make(map[string]*sync.Once)
	}

	o, ok := sf.m[key]
	if !ok {
		o = new(sync.Once)
		sf.m[key] = o
	}

	defer o.Do(f)
}

func (sf *singleflight) Reset(key string) {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	delete(sf.m, key)
}

// func max(x, y int) int {
// 	if x > y {
// 		return x
// 	}
// 	return y
// }
