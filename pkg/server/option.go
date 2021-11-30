package server

import (
	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/pnet"
	"github.com/lthibault/log"
	ma "github.com/multiformats/go-multiaddr"
)

// Option type for Node
type Option func(*Node)

func WithLogger(l log.Logger) Option {
	if l == nil {
		l = log.New()
	}

	return func(n *Node) {
		n.log = l
	}
}

func WithNamepace(ns string) Option {
	if ns == "" {
		ns = "ww"
	}

	return func(n *Node) {
		n.ns = ns
	}
}

func WithSecret(s pnet.PSK) Option {
	return func(n *Node) {
		n.host.SetSecret(s)
	}
}

func WithAuth(auth connmgr.ConnectionGater) Option {
	return func(n *Node) {
		n.host.SetAuth(auth)
	}
}

func WithListenAddrs(ms ...ma.Multiaddr) Option {
	ss := make([]string, len(ms))
	for i, m := range ms {
		ss[i] = m.String()
	}

	return WithListenAddrStrings(ss...)
}

func WithListenAddrStrings(ss ...string) Option {
	return func(n *Node) {
		n.host.SetListenAddrs(ss...)
	}
}

func WithHost(f HostFactory) Option {
	if f == nil {
		f = &RoutedHostFactory{}
	}

	return func(n *Node) {
		n.host = f
	}
}

func WithDHT(f DHTFactory) Option {
	if f == nil {
		f = DualDHTFactory(nil)
	}

	return func(n *Node) {
		n.dht = f
	}
}

func WithPubSub(f PubSubFactory) Option {
	if f == nil {
		f = &GossipsubFactory{}
	}

	return func(n *Node) {
		n.ps = f
	}
}

func withDefaults(opt []Option) []Option {
	return append([]Option{
		WithNamepace(""),
		WithLogger(nil),
		WithHost(nil),
		WithDHT(nil),
		WithPubSub(nil),
	}, opt...)
}
