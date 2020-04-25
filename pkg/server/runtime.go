package server

/*
	runtime.go implements Host background processes
*/

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/lthibault/jitterbug"
	log "github.com/lthibault/log/pkg"
	"github.com/multiformats/go-multiaddr"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"

	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/event"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	ww "github.com/lthibault/wetware/pkg"
	"github.com/lthibault/wetware/pkg/boot"
	randutil "github.com/lthibault/wetware/pkg/util/rand"
)

const (
	uagentKey      = "AgentVersion"
	tagStreamInUse = "ww-stream-in-use"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type process func(runtime) fx.Hook

type runtime struct {
	fx.In

	Log log.Logger
	Ctx context.Context

	Host         host.Host
	ConnMgr      connmgr.ConnManager
	DHT          routingDiscovery
	BootProtocol boot.Protocol

	Namespace string
	TTL       time.Duration
	Cluster   *struct{ *pubsub.Topic }
}

// run is intended to be invoked from an Fx application.  It starts background processes
// for a given host.
func run(lx fx.Lifecycle, r runtime) {
	for _, proc := range []process{
		bootstrap,
		signalNetworkEvents,
		signalNeighborhoodEvents,
		protectConns,
		maintainNeighborhood,
		announcePresence,
	} {
		lx.Append(proc(r))
	}
}

func bootstrap(r runtime) fx.Hook {
	return fx.Hook{
		OnStart: func(context.Context) error {
			return r.BootProtocol.Start(r.Host)
		},
		OnStop: func(context.Context) error {
			return r.BootProtocol.Close()
		},
	}
}

// signalNetworkEvents hooks into the host's network and emits events over event.Bus to
// signal changes in connections or streams.
//
// HACK:  This is a short-term solution while we wait for libp2p to provide equivalent
//		  functionality.
func signalNetworkEvents(r runtime) fx.Hook {
	w := netEventWatcher{
		log:  r.Log,
		host: r.Host,
	}

	callbacks := &network.NotifyBundle{
		ConnectedF:    w.onConnected,
		DisconnectedF: w.onDisconnected,

		OpenedStreamF: w.onStreamOpened,
		ClosedStreamF: w.onStreamOpened,
	}

	return fx.Hook{
		OnStart: func(context.Context) error {
			w.host.Network().Notify(callbacks)
			return w.init()
		},
		OnStop: func(context.Context) error {
			return w.connEvt.Close()
		},
	}
}

type netEventWatcher struct {
	log              log.Logger
	host             host.Host
	connEvt, strmEvt event.Emitter
}

func (w *netEventWatcher) init() (err error) {
	if w.connEvt, err = w.host.EventBus().Emitter(new(ww.EvtConnectionChanged)); err != nil {
		return
	}

	if w.connEvt, err = w.host.EventBus().Emitter(new(ww.EvtStreamChanged)); err != nil {
		return
	}

	return
}

func (w netEventWatcher) onConnected(net network.Network, conn network.Conn) {
	w.connEvt.Emit(ww.EvtConnectionChanged{
		Peer:   conn.RemotePeer(),
		Client: isClient(conn.RemotePeer(), w.host.Peerstore()),
		State:  ww.ConnStateOpened,
	})
}

func (w netEventWatcher) onDisconnected(net network.Network, conn network.Conn) {
	w.connEvt.Emit(ww.EvtConnectionChanged{
		Peer:   conn.RemotePeer(),
		Client: isClient(conn.RemotePeer(), w.host.Peerstore()),
		State:  ww.ConnStateClosed,
	})
}

func (w netEventWatcher) onStreamOpened(net network.Network, s network.Stream) {
	w.strmEvt.Emit(ww.EvtStreamChanged{
		Peer:   s.Conn().RemotePeer(),
		Stream: s,
		State:  ww.StreamStateOpened,
	})
}

func (w netEventWatcher) onStreamClosed(net network.Network, s network.Stream) {
	w.strmEvt.Emit(ww.EvtStreamChanged{
		Peer:   s.Conn().RemotePeer(),
		Stream: s,
		State:  ww.StreamStateClosed,
	})
}

func signalNeighborhoodEvents(r runtime) fx.Hook {
	var sub event.Subscription

	return fx.Hook{
		OnStart: func(context.Context) (err error) {
			bus := r.Host.EventBus()
			if sub, err = bus.Subscribe(new(ww.EvtConnectionChanged)); err != nil {
				return
			}

			var e event.Emitter
			if e, err = bus.Emitter(new(ww.EvtNeighborhoodChanged)); err == nil {
				go neighborhoodEventLoop(sub, e)
			}

			return
		},
		OnStop: func(context.Context) error {
			return sub.Close()
		},
	}
}

func neighborhoodEventLoop(sub event.Subscription, e event.Emitter) {
	defer e.Close()

	var (
		fire bool
		out  ww.EvtNeighborhoodChanged
		n    neighborhood = make(map[peer.ID]int)
	)

	for v := range sub.Out() {
		ev := v.(ww.EvtConnectionChanged)
		if ev.Client {
			continue
		}

		switch ev.State {
		case ww.ConnStateOpened:
			fire = n.Add(ev.Peer)
		case ww.ConnStateClosed:
			fire = n.Rm(ev.Peer)
		default:
			panic(fmt.Sprintf("unknown conn state %d", ev.State))
		}

		if fire {
			out = ww.EvtNeighborhoodChanged{
				Peer:  ev.Peer,
				State: ev.State,
				From:  out.To,
				To:    n.Phase(),
				N:     len(n),
			}

			e.Emit(out)
		}
	}
}

func protectConns(r runtime) fx.Hook {
	var sub event.Subscription

	return fx.Hook{
		OnStart: func(context.Context) (err error) {
			if sub, err = r.Host.EventBus().Subscribe([]interface{}{
				new(ww.EvtNeighborhoodChanged),
				new(ww.EvtStreamChanged),
			}); err == nil {
				go connProtectorLoop(sub, r.ConnMgr)
			}

			return
		},
		OnStop: func(context.Context) error {
			return sub.Close()
		},
	}
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
	ns  string

	host host.Host

	sf singleflight
	b  boot.Strategy
	d  discovery.Discoverer
}

func maintainNeighborhood(r runtime) fx.Hook {
	var sub event.Subscription
	m := neighborhoodMaintainer{
		log:  r.Log,
		ns:   r.Namespace,
		host: r.Host,
		b:    r.BootProtocol,
		d:    r.DHT,
	}

	return fx.Hook{
		OnStart: func(context.Context) (err error) {
			if sub, err = m.host.EventBus().
				Subscribe(new(ww.EvtNeighborhoodChanged)); err == nil {
				go m.loop(r.Ctx, sub)
			}

			return
		},
		OnStop: func(context.Context) error {
			return sub.Close()
		},
	}

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
			m.graft(reqctx, max((ww.LowWater-ev.N)/2, 1))
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

		var g errgroup.Group
		for _, pinfo := range ps {
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

func connect(ctx context.Context, host host.Host, pinfo peer.AddrInfo) func() error {
	return func() error {
		ctx, cancel := context.WithTimeout(ctx, time.Second*15)
		defer cancel()

		return host.Connect(ctx, pinfo)
	}
}

type announcer struct {
	log log.Logger

	ttl  time.Duration
	host interface {
		ID() peer.ID
		Addrs() []multiaddr.Multiaddr
	}
	mesh interface {
		Publish(context.Context, []byte, ...pubsub.PubOpt) error
	}
}

func announcePresence(r runtime) fx.Hook {
	ctx, cancel := context.WithCancel(r.Ctx)
	return fx.Hook{
		OnStart: func(start context.Context) (err error) {
			a := announcer{
				log:  r.Log,
				host: r.Host,
				ttl:  r.TTL,
				mesh: r.Cluster.Topic,
			}

			if err = a.Announce(start); err == nil {
				go a.loop(ctx)
			}

			return
		},
		OnStop: func(stop context.Context) error {
			cancel()
			return nil
		},
	}
}

func (a announcer) Announce(ctx context.Context) error {
	hb, err := ww.NewHeartbeat(peer.AddrInfo{
		ID:    a.host.ID(),
		Addrs: a.host.Addrs(),
	}, a.ttl)
	if err != nil {
		return err
	}

	b, err := ww.MarshalHeartbeat(hb)
	if err != nil {
		return err
	}

	return a.mesh.Publish(ctx, b)
}

func (a announcer) loop(ctx context.Context) {
	// Hosts tend to be started in batches, which causes heartbeat storms.  We
	// add a small ammount of jitter to smooth things out.  The jitter is
	// calculated by sampling from a uniform distribution between .25 * TTL and
	// .5 * TTL.  The TTL corresponds to 2.6 heartbeats, on average.
	//
	// With default TTL settings, a heartbeat is emitted every 2250ms, on
	// average.  This tolerance is optimized for the widest possible variety of
	// execution settings, and should notably perform well on high-latency
	// networks, including 3G.
	//
	// Clusters operating in low-latency settings such as datacenters may wish
	// to reduce the TTL.  Doing so will increase the cluster's responsiveness
	// at the expense of an O(n) increase in bandwidth consumption.
	ticker := jitterbug.New(a.ttl/2, jitterbug.Uniform{
		Min:    a.ttl / 4,
		Source: rand.New(randutil.FromPeer(a.host.ID())),
	})
	defer ticker.Stop()

	for range ticker.C {
		switch err := a.Announce(ctx); err {
		case nil:
		case context.Canceled:
			return
		default:
			a.log.WithError(err).Error("failed to announce")
		}
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

type neighborhood map[peer.ID]int

func (n neighborhood) Add(id peer.ID) (leased bool) {
	i, ok := n[id]
	if !ok {
		leased = true
	}

	n[id] = i + 1
	return
}

func (n neighborhood) Rm(id peer.ID) (evicted bool) {
	// ok == false implies a client disconnected
	if i, ok := n[id]; ok && i == 1 {
		delete(n, id)
		evicted = true
	}

	return
}

func (n neighborhood) Phase() ww.Phase {
	switch k := len(n); {
	case k < 0:
		return ww.PhaseOrphaned
	case k < ww.LowWater:
		return ww.PhasePartial
	case k < ww.HighWater:
		return ww.PhaseComplete
	case k >= ww.HighWater:
		return ww.PhaseOverloaded
	default:
		panic(fmt.Sprintf("invalid number of connections: %d", k))
	}
}

func isClient(p peer.ID, ps peerstore.PeerMetadata) bool {
	v, err := ps.Get(p, uagentKey)
	if err == nil {
		return v.(string) == ww.ClientUAgent
	}

	panic(err)
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
