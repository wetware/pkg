package host

import (
	"context"

	"go.uber.org/fx"

	"github.com/davecgh/go-spew/spew"
	"github.com/libp2p/go-libp2p-core/event"
	host "github.com/libp2p/go-libp2p-core/host"
)

func eventloop(ctx context.Context, host host.Host) fx.Hook {
	bus := eventbus{
		host: host,
	}

	return fx.Hook{
		OnStart: func(context.Context) error {
			return bus.Start()
		},
		OnStop: func(context.Context) error {
			return bus.Close()
		},
	}
}

type eventbus struct {
	host host.Host
	sub  event.Subscription
}

func (bus *eventbus) Start() (err error) {
	if bus.sub, err = bus.host.EventBus().Subscribe([]interface{}{
		new(event.EvtPeerConnectednessChanged),
	}); err == nil {
		go bus.loop()
	}

	return
}

func (bus eventbus) Close() error {
	return bus.sub.Close()
}

func (bus eventbus) loop() {
	for v := range bus.sub.Out() {
		switch e := v.(type) {
		case event.EvtPeerConnectednessChanged:
			for _, conn := range bus.host.Network().ConnsToPeer(e.Peer) {
				spew.Dump(conn.Stat().Extra) // DEBUG
			}
		}
	}
}

// func bundle(hs ...fx.Hook) fx.Hook {
// 	return fx.Hook{
// 		OnStart: hookStart(hs),
// 		OnStop:  hookStop(hs),
// 	}
// }

// func hookStart(hs []fx.Hook) func(context.Context) error {
// 	return func(ctx context.Context) error {
// 		starters := make([]func(context.Context) error, len(hs))
// 		for i, hook := range hs {
// 			starters[i] = hook.OnStart
// 		}

// 		return hookFuncs(ctx, starters)
// 	}
// }

// func hookStop(hs []fx.Hook) func(context.Context) error {
// 	return func(ctx context.Context) error {
// 		stoppers := make([]func(context.Context) error, len(hs))
// 		for i, hook := range hs {
// 			stoppers[i] = hook.OnStop
// 		}

// 		return hookFuncs(ctx, stoppers)
// 	}
// }

// func hookFuncs(ctx context.Context, fs []func(context.Context) error) error {
// 	g, ctx := errgroup.WithContext(ctx)
// 	for _, f := range fs {
// 		g.Go(hookFunc(ctx, f))
// 	}
// 	return g.Wait()
// }

// func hookFunc(ctx context.Context, f func(context.Context) error) func() error {
// 	return func() error {
// 		return f(ctx)
// 	}
// }
