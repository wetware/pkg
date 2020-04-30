package server

import (
	"context"
	"sync"
	"time"

	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"

	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/event"
	host "github.com/libp2p/go-libp2p-core/host"
	discovery "github.com/libp2p/go-libp2p-discovery"
	log "github.com/lthibault/log/pkg"
	discover "github.com/lthibault/wetware/pkg/discover"

	ww "github.com/lthibault/wetware/pkg"
)

/*
	connmgr.go manages host connections through (a) the libp2p `ConnManager`
	interface and (b) the ww `neighborhood` object.
*/

const tagStreamInUse = "ww-stream-in-use"

type connmgrConfig struct {
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

func connmanager(lx fx.Lifecycle, cfg connmgrConfig) error {
	if err := protectConns(lx, cfg.Host, cfg.ConnMgr); err != nil {
		return err
	}

	sub, err := cfg.Host.EventBus().Subscribe(new(ww.EvtNeighborhoodChanged))
	if err != nil {
		return err
	}
	lx.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return sub.Close()
		},
	})

	m := &neighborhoodMaintainer{
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
}

func protectConns(lx fx.Lifecycle, host host.Host, cm connmgr.ConnManager) error {
	sub, err := host.EventBus().Subscribe([]interface{}{
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

	go connProtectorLoop(sub, cm)
	return nil
}

func connProtectorLoop(sub event.Subscription, cm connmgr.ConnManager) {
	for v := range sub.Out() {
		switch ev := v.(type) {
		case ww.EvtNeighborhoodChanged:
			switch ev.State {
			case ww.ConnStateOpened:
				// TODO:  ... What's our policy for protecting Host connections?
				panic("function NOT IMPLEMENTED")
			case ww.ConnStateClosed:
				cm.UntagPeer(ev.Peer, tagStreamInUse)
			}
		case ww.EvtStreamChanged:
			switch ev.State {
			case ww.StreamStateOpened:
				cm.TagPeer(ev.Peer, tagStreamInUse, 1)
			case ww.StreamStateClosed:
				cm.TagPeer(ev.Peer, tagStreamInUse, -1)
			}
		}

	}
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
			m.graft(reqctx, max((m.kmin-ev.N)/2, 1))
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
			continue
		case v, ok := <-sub.Out():
			if !ok {
				cancel()
				return
			}

			ev = v.(ww.EvtNeighborhoodChanged)
		}
	}
}

func (m *neighborhoodMaintainer) join(ctx context.Context) {
	go m.sf.Do("join", func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second*30)
		defer cancel()
		defer m.sf.Reset("join")

		ps, err := m.b.DiscoverPeers(ctx)
		if err != nil {
			m.log.WithError(err).Debug("peer discovery failed")
		}

		self := m.host.ID()
		var g errgroup.Group
		for _, pinfo := range ps {
			if pinfo.ID == self {
				continue // got our own addr info; skip.
			}

			g.Go(connect(ctx, m.host, pinfo))
		}

		if err = g.Wait(); err != nil {
			m.log.WithError(err).Debug("peer connection failed")
		}
	})
}

func (m *neighborhoodMaintainer) graft(ctx context.Context, limit int) {
	go m.sf.Do("graft", func() {
		discoverCtx, cancel := context.WithTimeout(ctx, time.Second*30)
		defer cancel()
		defer m.sf.Reset("graft")

		ch, err := m.d.FindPeers(discoverCtx, m.ns, discovery.Limit(limit))
		if err != nil {
			m.log.WithError(err).Debug("discovery failed")
			return
		}

		var g errgroup.Group
		for pinfo := range ch {
			g.Go(connect(ctx, m.host, pinfo))
		}

		if err = g.Wait(); err != nil {
			m.log.WithError(err).Debug("peer connection failed")

		}
	})
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

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
