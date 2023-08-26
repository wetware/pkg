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

// type Server[T ~capnp.ClientKind] struct {
// 	*closer
// 	host.Server
// 	*cluster.Router
// }

// func (s Server[T]) Serve(ctx context.Context, h local.Host, cfg Config[T]) error {
// 	conf.export(ctx, h, s)

// 	if err := s.Bootstrap(ctx); err != nil {
// 		return fmt.Errorf("bootstrap: %w", err)
// 	}

// 	conf.Logger.Info("wetware started")
// 	defer conf.Logger.Warn("wetware stopped")

// 	<-ctx.Done()
// 	return ctx.Err()
// }

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

	ns := &boot.Namespace{
		Name:      conf.NS,
		Bootstrap: conf.bootstrap(),
		Ambient:   conf.ambient(dht),
	}

	pubsub, err := pubsub.NewGossipSub(context.TODO(), h,
		pubsub.WithPeerExchange(true),
		// pubsub.WithRawTracer(conf.tracer()),
		pubsub.WithDiscovery(ns),
		pubsub.WithProtocolMatchFn(conf.protoMatchFunc()),
		pubsub.WithGossipSubProtocols(conf.subProtos()),
		pubsub.WithPeerOutboundQueueSize(1024),
		pubsub.WithValidateQueueSize(1024),
	)
	if err != nil {
		return failuref("pubsub: %w", err)
	}

	cluster, err := cluster.Config{
		Net:    ns,
		Host:   h,
		PubSub: pubsub,
		Meta:   conf.Meta,
	}.Join(context.TODO())
	if err != nil {
		return failuref("cluster: %w", err)
	}
	closer = closer.push(cluster)

	server := host.Server{
		ViewProvider: cluster,
		TopicJoiner:  pubsub,
		RuntimeConfig: wazero.NewRuntimeConfigCompiler().
			WithCompilationCache(wazero.NewCompilationCache()).
			WithCloseOnContextDone(true),
	}

	return server.Host(), closer
}

func failure(err error) (host.Host, io.Closer) {
	return host.Host(capnp.ErrorClient(err)), io.NopCloser(nil)
}

func failuref(format string, args ...any) (host.Host, io.Closer) {
	return failure(fmt.Errorf(format, args...))
}

// func (cfg Config[T]) ListenAndServe(ctx context.Context, addrs ...string) error {
// 	h, err := libp2p.New(
// 		libp2p.NoTransports,
// 		libp2p.Transport(tcp.NewTCPTransport),
// 		libp2p.Transport(quic.NewTransport),
// 		libp2p.ListenAddrStrings(addrs...))
// 	if err != nil {
// 		return fmt.Errorf("listen: %w", err)
// 	}
// 	defer h.Close()

// 	return conf.Serve(ctx, h)
// }

// func (cfg Config[T]) Serve(ctx context.Context, h local.Host) error {
// 	server, err := conf.NewServer(ctx, h)
// 	if err != nil {
// 		return err
// 	}
// 	defer server.Close()

// 	return server.Serve(ctx, h, cfg)
// }

// func (cfg Config[T]) NewServer(ctx context.Context, h local.Host) (*Server, error) {
// 	if conf.Logger == nil {
// 		conf.Logger = slog.Default()
// 	}

// 	var closer *closer

// 	d, err := conf.newBootstrapper(h)
// 	if err != nil {
// 		return nil, fmt.Errorf("discovery: %w", err)
// 	}
// 	closer = closer.push(d)

// 	h, dht, err := conf.withRouting(ctx, h)
// 	if err != nil {
// 		return nil, fmt.Errorf("dht: %w", err)
// 	}
// 	closer = closer.push(dht)

// 	ps, err := conf.newPubSub(ctx, pubSubConfig{
// 		Logger:    conf.Logger,
// 		NS:        conf.NS,
// 		Host:      h,
// 		Discovery: d,
// 		DHT:       dht,
// 	})
// 	if err != nil {
// 		return nil, fmt.Errorf("pubsub: %w", err)
// 	}

// 	c, err := conf.newCluster(ctx, clusterConfig{
// 		Host:      h,
// 		PubSub:    ps,
// 		Discovery: d,
// 		DHT:       dht,
// 	})
// 	if err != nil {
// 		return nil, fmt.Errorf("cluster: %w", err)
// 	}
// 	closer = closer.push(closeFunc(func() error {
// 		defer c.Stop()
// 		return ps.UnregisterTopicValidator(conf.NS)
// 	}))

// 	e, err := conf.newExecutor(ctx, executorConfig{
// 		Cache: make(csp_server.BytecodeCache),
// 		RuntimeCfg: wazero.
// 			NewRuntimeConfigCompiler().
// 			WithCompilationCache(wazero.NewCompilationCache()).
// 			WithCloseOnContextDone(true),
// 	})
// 	if err != nil {
// 		return nil, fmt.Errorf("executor: %w", err)
// 	}

// 	server := host.Server{
// 		ViewProvider:   c,
// 		AnchorProvider: &anchor.Node{},
// 		PubSubProvider: &pubsub.Server{
// 			Log:         conf.Logger,
// 			TopicJoiner: ps,
// 		},
// 		ExecutorProvider: e,
// 		CapStoreProvider: &capstore_server.CapStore{},
// 	}

// 	return &Server{
// 		closer: closer,
// 		Server: server,
// 		Router: c,
// 	}, nil
// }

// func (cfg Config[T]) export(ctx context.Context, h local.Host, boot ClientProvider) {
// 	for _, proto := range proto.Namespace(conf.NS) {
// 		h.SetStreamHandler(proto, func(s network.Stream) {
// 			defer s.Close()

// 			conn := rpc.NewConn(transport(s), &rpc.Options{
// 				ErrorReporter:   system.ErrorReporter{Logger: conf.Logger},
// 				BootstrapClient: boot.Client(), // host.Host
// 			})
// 			defer conn.Close()

// 			select {
// 			case <-ctx.Done():
// 			case <-conn.Done():
// 			}
// 		})
// 	}
// }

// func transport(s network.Stream) rpc.Transport {
// 	if strings.HasSuffix(string(s.Protocol()), "/packed") {
// 		return rpc.NewPackedStreamTransport(s)
// 	}

// 	return rpc.NewStreamTransport(s)
// }

// closer is a stack of io.Closers
type closer struct {
	closer io.Closer
	next   *closer
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
