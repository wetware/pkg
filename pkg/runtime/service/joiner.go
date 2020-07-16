package service

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/lthibault/wetware/pkg/runtime"
)

// Joiner performs a JOIN operation against the cluster graph, resulting in the merger
// of the local peer's graph and the remote peer's graph.
//
// Consumes:
//	- p2p.EvtNetworkReady
// 	- EvtPeerDiscovered
func Joiner(h host.Host) ProviderFunc {
	return func() (_ runtime.Service, err error) {
		ctx, cancel := context.WithCancel(context.Background())
		j := &joiner{
			h:      h,
			ctx:    ctx,
			cancel: cancel,
			errs:   make(chan error, 1),
			join:   make(chan EvtPeerDiscovered),
		}

		if j.sub, err = j.h.EventBus().Subscribe(new(EvtPeerDiscovered)); err != nil {
			return
		}

		return j, nil
	}
}

type joiner struct {
	h host.Host

	ctx    context.Context
	cancel context.CancelFunc

	errs chan error
	join chan EvtPeerDiscovered
	sub  event.Subscription
}

// Loggable representation
func (j joiner) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"service": "joiner",
		"host":    j.h.ID(),
	}
}

// Start service
func (j *joiner) Start(ctx context.Context) (err error) {
	if err = waitNetworkReady(ctx, j.h.EventBus()); err == nil {
		startBackground(
			j.subloop,
			j.joinloop,
		)
	}

	return
}

// Stop service
func (j joiner) Stop(ctx context.Context) error {
	defer j.cancel()

	return j.sub.Close()
}

// Errors .
func (j joiner) Errors() <-chan error {
	return j.errs
}

func (j joiner) subloop() {
	defer close(j.join)

	for v := range j.sub.Out() {
		select {
		case j.join <- v.(EvtPeerDiscovered):
		case <-j.ctx.Done():
		default:
			// there's already a join in progress
		}
	}
}

func (j joiner) joinloop() {
	defer close(j.errs)

	for ev := range j.join {
		if ev.ID == j.h.ID() {
			continue
		}

		j.connect(peer.AddrInfo(ev))
	}
}

func (j joiner) connect(info peer.AddrInfo) {
	ctx, cancel := context.WithTimeout(j.ctx, time.Second*30)
	defer cancel()

	j.raise(j.h.Connect(ctx, info))
}

func (j joiner) raise(err error) {
	if err == nil {
		return
	}

	select {
	case j.errs <- err:
	case <-j.ctx.Done():
	}
}
