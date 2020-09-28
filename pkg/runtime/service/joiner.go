package service

import (
	"context"
	"time"

	"github.com/lthibault/log"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"

	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/runtime"
)

// Joiner performs a JOIN operation against the cluster graph, resulting in the merger
// of the local peer's graph and the remote peer's graph.
//
// Consumes:
//	- p2p.EvtNetworkReady
// 	- EvtPeerDiscovered
func Joiner(log log.Logger, h host.Host) ProviderFunc {
	return func() (_ runtime.Service, err error) {
		ctx, cancel := context.WithCancel(context.Background())
		j := &joiner{
			log:    log,
			h:      h,
			ctx:    ctx,
			cancel: cancel,
			join:   make(chan EvtPeerDiscovered, 1),
		}

		if j.sub, err = j.h.EventBus().Subscribe(new(EvtPeerDiscovered)); err != nil {
			return
		}

		return j, nil
	}
}

type joiner struct {
	log ww.Logger
	h   host.Host

	ctx    context.Context
	cancel context.CancelFunc

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
	for ev := range j.join {
		// NOTE:  It is important that the joiner respond to _ALL_ events it receives
		//		  on this channel.  Event validation (e.g. to avoid a connect-to-self)
		//		  must take place at the source, else a sublte race condition may occur.
		//
		//		  The race condition works like this:
		//		   1. emit EvtPeerDiscovered
		//		   2. joiner receives event, compares to local host's ID, drops event.
		//		   3. in the meantime, subsequent (valid) events arrive, but joinloop is
		//			  not receiving, so the events are dropped.
		j.connect(peer.AddrInfo(ev))
	}
}

func (j joiner) connect(info peer.AddrInfo) {
	ctx, cancel := context.WithTimeout(j.ctx, time.Second*30)
	defer cancel()

	if err := j.h.Connect(ctx, info); err != nil {
		j.log.With(j).
			WithError(err).
			Debugf("unable to connct to %s", info.ID)
	}
}
