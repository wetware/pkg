package server

import (
	"context"
	"fmt"
	"io"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/discovery"
	local "github.com/libp2p/go-libp2p/core/host"
	"go.uber.org/multierr"

	"capnproto.org/go/capnp/v3"
	"github.com/tetratelabs/wazero"

	"github.com/wetware/pkg/boot"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/cluster"
)

type Config struct {
	NS        string
	Host      local.Host
	Meta      []string
	Discovery discovery.Discovery
}

func (conf Config) Client() (host.Host, io.Closer) {
	var closer *closer

	h, dht, err := conf.withRouting()
	if err != nil {
		return failuref("dht: %w", err)
	}
	closer = closer.push(dht)

	d := &boot.Namespace{
		Name:      conf.NS,
		Bootstrap: conf.bootstrap(),
		Ambient:   conf.ambient(dht),
	}

	pubsub, err := pubsub.NewGossipSub(context.TODO(), h,
		pubsub.WithPeerExchange(true),
		// pubsub.WithRawTracer(conf.tracer()),
		pubsub.WithDiscovery(d),
		pubsub.WithProtocolMatchFn(conf.protoMatchFunc()),
		pubsub.WithGossipSubProtocols(conf.subProtos()),
		pubsub.WithPeerOutboundQueueSize(1024),
		pubsub.WithValidateQueueSize(1024))
	if err != nil {
		return failuref("pubsub: %w", err)
	}

	cluster, err := cluster.Config{
		NS:     conf.NS,
		Host:   h,
		PubSub: pubsub,
		Meta:   conf.Meta,
	}.Join(context.TODO())
	if err != nil {
		return failuref("cluster: %w", err)
	}
	closer = closer.push(cluster)

	if err := cluster.Bootstrap(context.TODO()); err != nil {
		return failuref("bootstrap: %w", err)
	}

	server := host.Server{
		ViewProvider: cluster,
		TopicJoiner:  pubsub,
		RuntimeConfig: wazero.NewRuntimeConfigCompiler().
			WithCompilationCache(wazero.NewCompilationCache()).
			WithCloseOnContextDone(true),
	}

	return server.Host(), closer
}

// closer is a stack of io.Closers
type closer struct {
	closer io.Closer
	next   *closer
}

func failure(err error) (host.Host, io.Closer) {
	return host.Host(capnp.ErrorClient(err)), io.NopCloser(nil)
}

func failuref(format string, args ...any) (host.Host, io.Closer) {
	return failure(fmt.Errorf(format, args...))
}

func (tail *closer) push(c io.Closer) *closer {
	return &closer{
		closer: c,
		next:   tail,
	}
}

func (tail *closer) Close() error {
	if tail == nil {
		return nil
	}

	return multierr.Combine(
		tail.closer.Close(),
		tail.next.Close())
}
