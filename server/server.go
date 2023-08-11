package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/libp2p/go-libp2p"
	local_host "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/tetratelabs/wazero"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"

	"github.com/wetware/pkg/cap/anchor"
	"github.com/wetware/pkg/cap/auth"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/cap/pubsub"
	"github.com/wetware/pkg/util/proto"
	"golang.org/x/exp/slog"
)

// Logger is used for logging by the RPC system. Each method logs
// messages at a different level, but otherwise has the same semantics:
//
//   - Message is a human-readable description of the log event.
//   - Args is a sequenece of key, value pairs, where the keys must be strings
//     and the values may be any type.
//   - The methods may not block for long periods of time.
//
// This interface is designed such that it is satisfied by *slog.Logger.
type Logger interface {
	Debug(message string, args ...any)
	Info(message string, args ...any)
	Warn(message string, args ...any)
	Error(message string, args ...any)
}

type Config struct {
	Logger   Logger
	NS       string
	Peers    []string // static bootstrap peers
	Discover string   // bootstrap service multiadr
	Meta     map[string]string
}

func (cfg Config) ListenAndServe(ctx context.Context, addrs ...string) error {
	h, err := libp2p.New(
		libp2p.NoTransports,
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(quic.NewTransport),
		libp2p.ListenAddrStrings(addrs...))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer h.Close()

	return cfg.Serve(ctx, h)
}

func (cfg Config) Serve(ctx context.Context, h local_host.Host) error {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	// cfg.Logger = cfg.Logger.WithField("ns", cfg.NS)

	d, err := cfg.newBootstrapper(h)
	if err != nil {
		return fmt.Errorf("discovery: %w", err)
	}
	defer d.Close()

	h, dht, err := cfg.withRouting(ctx, h)
	if err != nil {
		return fmt.Errorf("dht: %w", err)
	}
	defer dht.Close()

	ps, err := cfg.newPubSub(ctx, pubSubConfig{
		Logger:    cfg.Logger,
		NS:        cfg.NS,
		Host:      h,
		Discovery: d,
		DHT:       dht,
	})
	if err != nil {
		return fmt.Errorf("pubsub: %w", err)
	}

	c, err := cfg.newCluster(ctx, clusterConfig{
		Host:      h,
		PubSub:    ps,
		Discovery: d,
		DHT:       dht,
	})
	if err != nil {
		return fmt.Errorf("cluster: %w", err)
	}
	defer c.Stop()
	defer ps.UnregisterTopicValidator(cfg.NS)

	e, err := cfg.newExecutor(ctx, executorConfig{
		RuntimeCfg: wazero.
			NewRuntimeConfigCompiler().
			WithCompilationCache(wazero.NewCompilationCache()).
			WithCloseOnContextDone(true),
	})
	if err != nil {
		return fmt.Errorf("executor: %w", err)
	}

	cfg.export(ctx, h, &host.Server{
		ViewProvider:   c,
		AnchorProvider: &anchor.Node{},
		PubSubProvider: &pubsub.Server{
			Log:         cfg.Logger,
			TopicJoiner: ps,
		},
		ExecutorProvider: e,
	})

	if err := c.Bootstrap(ctx); err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}

	<-ctx.Done()
	return ctx.Err()
}

func (cfg Config) export(ctx context.Context, h local_host.Host, s *host.Server) {
	for _, proto := range proto.Namespace(cfg.NS) {
		h.SetStreamHandler(proto, cfg.handler(ctx, s))
	}
}

func (cfg Config) handler(ctx context.Context, h *host.Server) network.StreamHandler {
	return func(s network.Stream) {
		defer s.Close()

		conn := rpc.NewConn(transport(s), &rpc.Options{
			BootstrapClient: cfg.authProvider(h),
		})
		defer conn.Close()

		select {
		case <-ctx.Done():
		case <-conn.Done():
		}
	}
}

func (cfg Config) authProvider(h *host.Server) capnp.Client {
	policy := auth.AllowAll(h) // TODO(soon):  implement server-side auth here
	return capnp.Client(policy)
}

func transport(s network.Stream) rpc.Transport {
	if strings.HasSuffix(string(s.Protocol()), "/packed") {
		return rpc.NewPackedStreamTransport(s)
	}

	return rpc.NewStreamTransport(s)
}
