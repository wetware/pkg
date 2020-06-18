package runtime

import (
	"context"
	"sync"
	"time"

	log "github.com/lthibault/log/pkg"
	syncutil "github.com/lthibault/util/sync"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/lthibault/wetware/pkg/discover"
)

/*
	graph.go contains the logic responsible for ensuring cluster connectivity.  It
	it enacts a policy that attempts to maintain between kmin and kmax unique
	connections.
*/

// GraphParams .
type GraphParams struct {
	MinNeighbors, MaxNeighbors int
}

func (ps GraphParams) null() bool {
	return ps.MinNeighbors|ps.MaxNeighbors == 0
}

type graphBuilderParams struct {
	fx.In

	Log        log.Logger
	Namespace  string `name:"ns"`
	Graph      GraphParams
	Host       host.Host
	Boot       discover.Protocol
	PeerFinder discovery.Discovery
}

func buildGraph(ctx context.Context, ps graphBuilderParams, lx fx.Lifecycle) error {
	gb, err := newGraphBuilder(ps)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())

	lx.Append(fx.Hook{
		OnStart: func(context.Context) error {
			ps.Log.WithField("service", "graph").Debug("service started")
			go gb.loop(ctx)
			return nil
		},
		OnStop: func(context.Context) error {
			defer ps.Log.WithField("service", "graph").Debug("service stopped")
			cancel()
			return gb.Close()
		},
	})

	return nil
}

func newGraphBuilder(ps graphBuilderParams) (*graphBuilder, error) {
	sub, err := ps.Host.EventBus().Subscribe(new(EvtNeighborhoodChanged))
	if err != nil {
		return nil, err
	}

	return &graphBuilder{
		log:     ps.Log.WithField("service", "graph"),
		ns:      ps.Namespace,
		gp:      ps.Graph,
		host:    ps.Host,
		b:       ps.Boot,
		cluster: ps.PeerFinder, // libp2p2 discovery.Discovery
		sub:     sub,
	}, nil
}

type graphBuilder struct {
	log log.Logger

	ns string
	gp GraphParams

	host host.Host

	sf      singleflight
	b       discover.Strategy
	cluster discovery.Discoverer

	sub event.Subscription
}

func (g *graphBuilder) Close() error {
	return g.sub.Close()
}

func (g *graphBuilder) loop(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 15)
	defer ticker.Stop()

	var (
		ev     EvtNeighborhoodChanged
		reqctx context.Context
		cancel context.CancelFunc
	)

	for {
		switch ev.To {
		case PhaseOrphaned:
			reqctx, cancel = context.WithCancel(ctx)
			g.join(reqctx)
		case PhasePartial:
			reqctx, cancel = context.WithCancel(ctx)
			g.graft(reqctx, g.gp.MinNeighbors-ev.N)
		case PhaseOverloaded:
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
		case v, ok := <-g.sub.Out():
			if ok {
				ev = v.(EvtNeighborhoodChanged)
			}
		case <-ctx.Done():
			cancel()
			return
		}
	}
}

func (g *graphBuilder) join(ctx context.Context) {
	go g.sf.Do("join", func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second*30)
		defer cancel()
		defer g.sf.Reset("join")

		ps, err := g.b.DiscoverPeers(ctx,
			discover.WithLogger(g.log),
			discover.WithLimit(3))
		if err != nil {
			g.log.WithError(err).Debug("peer discovery failed")
			return
		}

		var any syncutil.Any
		for info := range ps {
			if info.ID == g.host.ID() {
				continue // got our own addr info; skip.
			}

			any.Go(g.connect(ctx, info))
		}

		// N.B.: error will b nil if no peers were found.
		if err = any.Wait(); err != nil {
			g.log.WithError(err).Debug("join failed")
		}
	})
}

// TODO:  this needs work.  seems like parts of this are broken ...
func (g *graphBuilder) graft(ctx context.Context, limit int) {
	go g.sf.Do("graft", func() {
		discoverCtx, cancel := context.WithTimeout(ctx, time.Second*30)
		defer cancel()
		defer g.sf.Reset("graft")

		// TODO:  provide value for `g.ns` in the DHT.
		ch, err := g.cluster.FindPeers(discoverCtx, g.ns, discovery.Limit(limit))
		if err != nil {
			g.log.WithError(err).Debug("discovery failed")
			return
		}

		var grp errgroup.Group
		for info := range ch {
			// TODO:  filter out self, and filter out already-connected peers.

			grp.Go(g.connect(ctx, info))
		}

		if err = grp.Wait(); err != nil {
			g.log.WithError(err).Debug("graft failed")
		}
	})
}

func (g *graphBuilder) connect(ctx context.Context, info peer.AddrInfo) func() error {
	return func() error {
		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		if err := g.host.Connect(ctx, info); err != nil {
			g.log.WithError(err).
				WithField("peer", info.ID).
				Trace("connection attempt failed")
			return err
		}

		g.log.WithField("peer", info.ID).
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
