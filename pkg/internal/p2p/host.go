package p2p

import (
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/multiformats/go-multiaddr"
)

// Listener is implemented by the `host.Host` returned from New.
// It is used internally to start listening for connections.
type Listener interface {
	// ListenAndServe ensures the host's startup sequence is properly synchronized, and
	// guarantees that a EvtLocalAddressesUpdated event has been fired before it returns.
	// Note that the event is guaranteed to have been emitted on the host's event bus, but
	// it does NOT guarantee that all listeners have processed it.
	Listen(...multiaddr.Multiaddr) error
}

type listenerHost struct {
	host.Host
	dht routing.Routing
	sig addrChangeSignaller
}

func wrapHost(h host.Host, dht routing.Routing) listenerHost {
	return listenerHost{
		Host: routedhost.Wrap(h, dht),
		dht:  dht,
		sig:  h.(addrChangeSignaller),
	}
}

func (l listenerHost) Listen(addrs ...multiaddr.Multiaddr) error {
	sub, err := l.EventBus().Subscribe(new(event.EvtLocalAddressesUpdated))
	if err != nil {
		return err
	}
	defer sub.Close()

	if err := l.Network().Listen(addrs...); err != nil {
		return err
	}

	// Ensure the host fires event.EvtLocalAddressUpdated immediately.
	l.sig.SignalAddressChange()

	// Best-effort attempt at ensuring the DHT is booted when `server.New` returns.
	// This appears to help avoid issues in one-off commands (e.g. `ww ls`) where
	// no peers are found because the DHT is not yet boostrapped.  On the other hand,
	// it MAY be responsible for the occasional deadlock when invoking such commands.
	// TODO(investigate)
	l.dht.Bootstrap(nil) // `dht.IpfsDHT.Bootstrap` discards the `ctx` param.

	<-sub.Out()
	return nil
}

// WARNING: this interface is unstable and may removed from basichost.BasicHost in the
// 		    future.  Hopefully this will only happen after they properly refactor Host
// 			setup.
type addrChangeSignaller interface {
	SignalAddressChange()
}
