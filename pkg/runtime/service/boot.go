package service

import (
	"context"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/peer"

	logutil "github.com/wetware/ww/internal/util/log"
	"github.com/wetware/ww/pkg/boot"
	"github.com/wetware/ww/pkg/runtime"
)

// EvtPeerDiscovered .
type EvtPeerDiscovered peer.AddrInfo

// Bootstrap performs bootstrap peer discovery.
//
// Consumes:
//	- EvtNeighborhoodChanged
//
// Emits:
//  - EvtPeerDiscovered
func Bootstrap(bus event.Bus, s boot.Strategy) ProviderFunc {
	return func() (_ runtime.Service, err error) {
		ctx, cancel := context.WithCancel(context.Background())
		b := &bootstrapper{
			s:        s,
			bus:      bus,
			ctx:      ctx,
			cancel:   cancel,
			errs:     make(chan error, 1),
			discover: make(chan struct{}, 1),
		}

		if b.sub, err = bus.Subscribe(new(EvtNeighborhoodChanged)); err != nil {
			return
		}

		if b.foundPeer, err = bus.Emitter(new(EvtPeerDiscovered)); err != nil {
			return
		}

		return b, nil
	}
}

type bootstrapper struct {
	s   boot.Strategy
	bus event.Bus

	ctx    context.Context
	cancel context.CancelFunc

	errs      chan error
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
	if err = waitNetworkReady(ctx, b.bus); err == nil {
		startBackground(
			b.subloop,
			b.queryloop,
		)
	}

	return
}

func (b bootstrapper) Stop(context.Context) error {
	defer b.cancel()

	return b.sub.Close()
}

func (b bootstrapper) Errors() <-chan error {
	return b.errs
}

func (b bootstrapper) subloop() {
	defer close(b.discover)

	for v := range b.sub.Out() {
		// Only use bootstrap discovery if the local node is orphaned.
		if notOrphaned(v.(EvtNeighborhoodChanged)) {
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
	defer close(b.errs)
	defer b.foundPeer.Close() // see b.emit()

	for range b.discover {
		ch, err := b.s.DiscoverPeers(b.ctx, boot.WithLimit(3))
		if err != nil {
			b.raise(err)
			continue
		}

		for info := range ch {
			b.emit(info)
		}
	}
}

func (b bootstrapper) emit(info peer.AddrInfo) {
	b.raise(b.foundPeer.Emit(EvtPeerDiscovered(info)))
}

func (b bootstrapper) raise(err error) {
	if err == nil {
		return
	}

	select {
	case b.errs <- err:
	case <-b.ctx.Done():
	}
}

func notOrphaned(ev EvtNeighborhoodChanged) bool {
	return ev.To != PhaseOrphaned
}
