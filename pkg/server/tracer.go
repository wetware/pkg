package server

import (
	"context"

	"go.uber.org/fx"

	"github.com/libp2p/go-libp2p-core/event"
	host "github.com/libp2p/go-libp2p-core/host"
	log "github.com/lthibault/log/pkg"

	ww "github.com/lthibault/wetware/pkg"
)

type tracerConfig struct {
	fx.In

	Log         log.Logger
	EnableTrace bool `name:"trace"`
	Host        host.Host
}

// log local events at Trace level.
func tracer(lx fx.Lifecycle, cfg tracerConfig) error {
	if !cfg.EnableTrace {
		return nil
	}

	sub, err := cfg.Host.EventBus().Subscribe([]interface{}{
		new(event.EvtLocalAddressesUpdated),
		new(event.EvtPeerIdentificationCompleted),
		new(event.EvtPeerIdentificationFailed),
		new(ww.EvtConnectionChanged),
		new(ww.EvtStreamChanged),
		new(ww.EvtNeighborhoodChanged),
	})
	if err != nil {
		return err
	}
	lx.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return sub.Close()
		},
	})

	go func() {
		tracer := cfg.Log.WithFields(log.F{
			"id":    cfg.Host.ID(),
			"addrs": cfg.Host.Addrs(),
		})

		tracer.Trace("event trace started")
		defer tracer.Trace("event trace finished")

		for v := range sub.Out() {
			switch ev := v.(type) {
			case event.EvtLocalAddressesUpdated:
				tracer = tracer.WithField("addrs", cfg.Host.Addrs())
				tracer.Trace("local addrs updated")
			case event.EvtPeerIdentificationCompleted:
				tracer.WithField("peer", ev.Peer).
					Trace("peer identification completed")
			case event.EvtPeerIdentificationFailed:
				tracer.WithError(ev.Reason).WithField("peer", ev.Peer).
					Trace("peer identification failed")
			case ww.EvtConnectionChanged:
				tracer.WithFields(log.F{
					"peer":       ev.Peer,
					"conn_state": ev.State,
					"client":     ev.Client,
				}).Trace("connection state changed")
			case ww.EvtStreamChanged:
				tracer.WithFields(log.F{
					"peer":         ev.Peer,
					"stream_state": ev.State,
					"proto":        ev.Stream.Protocol(),
				}).Trace("stream state changed")
			case ww.EvtNeighborhoodChanged:
				tracer.WithFields(log.F{
					"peer":       ev.Peer,
					"conn_state": ev.State,
					"from":       ev.From,
					"to":         ev.To,
					"n":          ev.N,
				}).Trace("neighborhood changed")
			}
		}
	}()

	return nil
}
