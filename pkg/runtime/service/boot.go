package service

import (
	"context"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/peer"

	logutil "github.com/lthibault/wetware/internal/util/log"
	"github.com/lthibault/wetware/pkg/boot"
	"github.com/lthibault/wetware/pkg/runtime"
)

// EvtPeerDiscovered .
type EvtPeerDiscovered struct {
	Peer peer.AddrInfo
}

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
			discover: make(chan struct{}),
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
		go b.subloop()
		go b.queryloop()
	}

	return
}

func (b bootstrapper) Stop(context.Context) error {
	defer close(b.errs)
	defer close(b.discover)
	defer b.cancel()

	return b.sub.Close()
}

func (b bootstrapper) Errors() <-chan error {
	return b.errs
}

func (b bootstrapper) subloop() {
	for v := range b.sub.Out() {
		if orphaned(v.(EvtNeighborhoodChanged)) {
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
	defer b.foundPeer.Close()

	for range b.discover {
		ch, err := b.s.DiscoverPeers(b.ctx, boot.WithLimit(3))
		if err != nil {
			b.errs <- err
			continue
		}

		for info := range ch {
			if err = b.emit(info); err != nil {
				b.errs <- err
			}
		}

	}
}

func (b bootstrapper) emit(info peer.AddrInfo) error {
	return b.foundPeer.Emit(EvtPeerDiscovered{Peer: info})
}

func orphaned(ev EvtNeighborhoodChanged) bool {
	return ev.To != PhaseOrphaned
}
