package service

import (
	"context"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/lthibault/wetware/pkg/runtime"
)

type joiner struct {
	h host.Host

	ctx    context.Context
	cancel context.CancelFunc

	errs chan error
	join chan EvtPeerDiscovered
	sub  event.Subscription
}

// Joiner performs a JOIN operation against the cluster graph, resulting in the merger
// of the local peer's graph and the remote peer's graph.
//
// Consumes:
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
		go j.subloop()
		go j.joinloop()
	}

	return
}

// Stop service
func (j joiner) Stop(ctx context.Context) error {
	defer close(j.errs)
	defer close(j.join)
	defer j.cancel()
	return j.sub.Close()
}

// Errors .
func (j joiner) Errors() <-chan error {
	return j.errs
}

func (j joiner) subloop() {
	for v := range j.sub.Out() {
		select {
		case j.join <- v.(EvtPeerDiscovered):
		default:
			// there's already a join in progress
		}
	}
}

func (j joiner) joinloop() {
	for ev := range j.join {
		if err := j.h.Connect(j.ctx, ev.Peer); err != nil {
			j.errs <- err
		}
	}
}
