package server

// const uagentKey = "AgentVersion"

// func eventloop(ctx context.Context, log log.Logger, host host.Host) fx.Hook {
// 	var sub event.Subscription

// 	return fx.Hook{
// 		OnStart: func(context.Context) (err error) {
// 			bus := newEventBus(log, host)

// 			if sub, err = host.EventBus().Subscribe([]interface{}{
// 				// new(ww.ConnectionEstablished),
// 			}); err == nil {
// 				go bus.loop(sub)
// 			}

// 			return
// 		},
// 		OnStop: func(context.Context) error {
// 			return sub.Close()
// 		},
// 	}
// }

// type eventbus struct {
// 	log  log.Logger
// 	meta peerstore.PeerMetadata
// }

// func newEventBus(log log.Logger, host host.Host) eventbus {
// 	bus := eventbus{log: log, meta: host.Peerstore()}
// 	host.Network().Notify(bus.Notifiee())
// 	return bus
// }

// func (bus eventbus) loop(sub event.Subscription) {
// 	for v := range sub.Out() {
// 		// switch e := v.(type) {
// 		// case ww.ConnectionEstablished:

// 		// }
// 	}
// }

// func (bus eventbus) Notifiee() *network.NotifyBundle {
// 	return &network.NotifyBundle{
// 		// ListenF      func(Network, ma.Multiaddr)
// 		// ListenCloseF func(Network, ma.Multiaddr)

// 		ConnectedF: bus.onConnected,
// 		// DisconnectedF: bus.onDisconnected,

// 		// OpenedStreamF func(Network, Stream)
// 		// ClosedStreamF func(Network, Stream)
// 	}
// }

// func (bus eventbus) onConnected(net network.Network, conn network.Conn) {
// 	v, err := bus.meta.Get(conn.RemotePeer(), uagentKey)
// 	if err != nil {
// 		bus.log.WithError(err).Error("failed to identify connection as host vs client.")
// 		conn.Close()
// 	}

// 	event := ww.ConnectionEstablished{ID: conn.RemotePeer()}

// 	switch uagent := v.(string); uagent {
// 	case ww.ClientUAgent:
// 		event.Type = ww.ConnTypeClient
// 	default:
// 		event.Type = ww.ConnTypeServer
// 		// it's a server; add it to the server set
// 	}
// }
