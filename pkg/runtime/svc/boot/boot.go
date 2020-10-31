package boot

import (
	"context"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"go.uber.org/fx"

	logutil "github.com/wetware/ww/internal/util/log"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/boot"
	"github.com/wetware/ww/pkg/runtime"
	"github.com/wetware/ww/pkg/runtime/svc/internal"
	"github.com/wetware/ww/pkg/runtime/svc/neighborhood"
)

// EvtPeerDiscovered .
type EvtPeerDiscovered peer.AddrInfo

// Config for Boot service.
type Config struct {
	fx.In

	Log      ww.Logger
	Host     host.Host
	Strategy boot.Strategy
}

// NewService satisfies runtime.ServiceFactory
func (cfg Config) NewService() (_ runtime.Service, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	b := &bootstrapper{
		log:      cfg.Log,
		s:        cfg.Strategy,
		h:        cfg.Host,
		ctx:      ctx,
		cancel:   cancel,
		discover: make(chan struct{}, 1),
	}

	if b.sub, err = cfg.Host.EventBus().Subscribe(new(neighborhood.EvtNeighborhoodChanged)); err != nil {
		return
	}

	if b.foundPeer, err = cfg.Host.EventBus().Emitter(new(EvtPeerDiscovered)); err != nil {
		return
	}

	return b, nil
}

// Produces EvtPeerDiscovered.
func (cfg Config) Produces() []interface{} {
	return []interface{}{
		EvtPeerDiscovered{},
	}
}

// Consumes neighborhood.EvtNeighborhoodChanged.
func (cfg Config) Consumes() []interface{} {
	return []interface{}{
		neighborhood.EvtNeighborhoodChanged{},
	}
}

// Module for Boot service.
type Module struct {
	fx.Out

	Factory runtime.ServiceFactory `group:"runtime"`
}

// New Boot service.  Performs bootstrap peer discovery.
func New(cfg Config) Module { return Module{Factory: cfg} }

type bootstrapper struct {
	log ww.Logger

	s boot.Strategy
	h host.Host

	ctx    context.Context
	cancel context.CancelFunc

	discover  chan struct{}
	sub       event.Subscription
	foundPeer event.Emitter
}

func (b bootstrapper) Loggable() map[string]interface{} {
	return logutil.JoinFields(
		map[string]interface{}{"service": "boot"},
		b.s.Loggable(),
	)
}

func (b *bootstrapper) Start(ctx context.Context) (err error) {
	if err = internal.WaitNetworkReady(ctx, b.h.EventBus()); err == nil {
		internal.StartBackground(
			b.queryloop,
			b.subloop,
		)
	}

	return
}

func (b bootstrapper) Stop(context.Context) error {
	defer b.cancel()

	return b.sub.Close()
}

func (b bootstrapper) subloop() {
	defer close(b.discover)

	for v := range b.sub.Out() {
		// Only use bootstrap discovery if the local node is orphaned.
		if notOrphaned(v.(neighborhood.EvtNeighborhoodChanged)) {
			continue
		}

		select {
		case b.discover <- struct{}{}:
		default:
			// there's already a join in progress
		}
	}
}

func (b bootstrapper) queryloop() {
	defer b.foundPeer.Close() // see b.emit()

	for range b.discover {
		ch, err := b.s.DiscoverPeers(b.ctx, boot.WithLimit(3))
		if err != nil {
			b.log.With(b).WithError(err).Debug("error discovering peers")
			continue
		}

		for info := range ch {
			if info.ID == b.h.ID() {
				continue
			}

			b.emit(info)
		}
	}
}

func (b bootstrapper) emit(info peer.AddrInfo) {
	if err := b.foundPeer.Emit(EvtPeerDiscovered(info)); err != nil {
		b.log.With(b).WithError(err).Error("failed to emit EvtPeerDiscovered")
	}
}

func notOrphaned(ev neighborhood.EvtNeighborhoodChanged) bool {
	return ev.To != neighborhood.PhaseOrphaned
}
