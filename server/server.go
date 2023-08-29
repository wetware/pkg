package server

import (
	"context"
	"fmt"

	"capnproto.org/go/capnp/v3"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"golang.org/x/exp/slog"

	"github.com/tetratelabs/wazero"

	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/boot"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/cluster"
)

type Config struct {
	NS    string
	Proto []protocol.ID
	Host  local.Host
	Boot  BootConfig
	Auth  auth.Policy[host.Host]
	Meta  []string
}

func (conf Config) Logger() *slog.Logger {
	return slog.Default().With(
		"ns", conf.NS,
		"peer", conf.Host.ID(),
		"proto", conf.Proto)
}

func (conf Config) Serve(ctx context.Context) error {
	h, dht, err := conf.withRouting()
	if err != nil {
		return fmt.Errorf("dht: %w", err)
	}
	defer dht.Close()

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
		return fmt.Errorf("pubsub: %w", err)
	}

	cluster, err := cluster.Config{
		NS:     conf.NS,
		Host:   h,
		PubSub: pubsub,
		Meta:   conf.Meta,
	}.Join(context.TODO())
	if err != nil {
		return fmt.Errorf("cluster: %w", err)
	}
	defer cluster.Close()

	if err := cluster.Bootstrap(context.TODO()); err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}

	server := host.Server{
		ViewProvider: cluster,
		TopicJoiner:  pubsub,
		RuntimeConfig: wazero.NewRuntimeConfigCompiler().
			WithCompilationCache(wazero.NewCompilationCache()).
			WithCloseOnContextDone(true),
	}

	host := server.Host()
	defer host.Release()

	release := conf.bind(ctx, h, host)
	defer release()

	conf.Logger().Info("wetware started")
	defer conf.Logger().Warn("wetware stopped")

	<-ctx.Done()
	return ctx.Err()
}

func (conf Config) bind(ctx context.Context, h local.Host, host host.Host) capnp.ReleaseFunc {
	for _, id := range conf.Proto {
		h.SetStreamHandler(id, conf.handler(ctx, h.ID(), host))
	}

	return func() {
		for _, id := range conf.Proto {
			h.RemoveStreamHandler(id)
		}
	}
}

func (conf Config) handler(ctx context.Context, id peer.ID, h host.Host) network.StreamHandler {
	return func(s network.Stream) {

	}
}
