package server

import (
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"
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

func WithTopics(ts ...string) Option {
	return func(n *Node) {
		n.ts = ts
	}
}

func WithHost(f HostFactory) Option {
	if f == nil {
		f = &RoutedHost{}
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

func WithBootStrategy(b BootStrategy) Option {
	if b == nil {
		b = &PortScanStrategy{}
	}

	return func(n *Node) {
		n.boot = b
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

func WithCluster(c ClusterConfig) Option {
	if c.NS == "" {
		c.NS = "ww"
	}

	if c.Ready == nil {
		c.Ready = pubsub.MinTopicSize(1)
	}

	return func(n *Node) {
		n.cc = c
	}
}

func withDefaults(opt []Option) []Option {
	return append([]Option{
		WithLogger(nil),
		WithHost(nil),
		WithBootStrategy(nil),
		WithDHT(nil),
		WithPubSub(nil),
	}, opt...)
}
