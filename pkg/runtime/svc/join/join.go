package joiner

import (
	"context"
	"time"

	"go.uber.org/fx"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"

	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/runtime"
	"github.com/wetware/ww/pkg/runtime/svc/boot"
	"github.com/wetware/ww/pkg/runtime/svc/internal"
)

// Config for Graph service.
type Config struct {
	fx.In

	Log  ww.Logger
	Host host.Host
}

// NewService satisfies runtime.ServiceFactory.
func (cfg Config) NewService() (_ runtime.Service, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	j := &joiner{
		log:    cfg.Log,
		h:      cfg.Host,
		ctx:    ctx,
		cancel: cancel,
		join:   make(chan boot.EvtPeerDiscovered, 1),
	}

	if j.sub, err = j.h.EventBus().Subscribe(new(boot.EvtPeerDiscovered)); err != nil {
		return
	}

	return j, nil
}

// Consumes boot.EvtPeerDiscovered.
func (cfg Config) Consumes() []interface{} {
	return []interface{}{
		boot.EvtPeerDiscovered{},
	}
}

// Module for Graph service.
type Module struct {
	fx.Out

	Factory runtime.ServiceFactory `group:"runtime"`
}

// New Joiner service.  Performs a JOIN operation against the cluster graph, resulting
// in the merger of the local peer's graph and the remote peer's graph.
//
// Consumes:
//	- p2p.EvtNetworkReady
// 	- EvtPeerDiscovered
func New(cfg Config) Module { return Module{Factory: cfg} }

type joiner struct {
	log ww.Logger
	h   host.Host

	ctx    context.Context
	cancel context.CancelFunc

	join chan boot.EvtPeerDiscovered
	sub  event.Subscription
}

// Loggable representation
func (j joiner) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"service": "join",
		"host":    j.h.ID(),
	}
}

// Start service
func (j *joiner) Start(ctx context.Context) (err error) {
	if err = internal.WaitNetworkReady(ctx, j.h.EventBus()); err == nil {
		internal.StartBackground(
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
		case j.join <- v.(boot.EvtPeerDiscovered):
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
