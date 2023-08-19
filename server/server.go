package server

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/libp2p/go-libp2p"
	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/tetratelabs/wazero"
	"go.uber.org/multierr"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"

	"github.com/wetware/pkg/cap/anchor"
	csp_server "github.com/wetware/pkg/cap/csp/server"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/cap/pubsub"
	"github.com/wetware/pkg/cluster"
	"github.com/wetware/pkg/system"
	"github.com/wetware/pkg/util/log"
	"github.com/wetware/pkg/util/proto"
	"golang.org/x/exp/slog"
)

type ClientProvider interface {
	Client() capnp.Client
}

type Server struct {
	*closer
	host.Server
	*cluster.Router
}

type Config struct {
	Logger   log.Logger
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

func (cfg Config) Serve(ctx context.Context, h local.Host) error {
	server, err := cfg.NewServer(ctx, h)
	if err != nil {
		return err
	}
	defer server.Close()

	cfg.export(ctx, h, server)

	if err := server.Bootstrap(ctx); err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}

	cfg.Logger.Info("wetware started")
	defer cfg.Logger.Warn("wetware stopped")

	<-ctx.Done()
	return ctx.Err()
}

func (cfg Config) NewServer(ctx context.Context, h local.Host) (*Server, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	var closer *closer

	d, err := cfg.newBootstrapper(h)
	if err != nil {
		return nil, fmt.Errorf("discovery: %w", err)
	}
	closer = closer.push(d)

	h, dht, err := cfg.withRouting(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("dht: %w", err)
	}
	closer = closer.push(dht)

	ps, err := cfg.newPubSub(ctx, pubSubConfig{
		Logger:    cfg.Logger,
		NS:        cfg.NS,
		Host:      h,
		Discovery: d,
		DHT:       dht,
	})
	if err != nil {
		return nil, fmt.Errorf("pubsub: %w", err)
	}

	c, err := cfg.newCluster(ctx, clusterConfig{
		Host:      h,
		PubSub:    ps,
		Discovery: d,
		DHT:       dht,
	})
	if err != nil {
		return nil, fmt.Errorf("cluster: %w", err)
	}
	closer = closer.push(closeFunc(func() error {
		defer c.Stop()
		return ps.UnregisterTopicValidator(cfg.NS)
	}))

	e, err := cfg.newExecutor(ctx, executorConfig{
		Cache: make(csp_server.BytecodeCache),
		RuntimeCfg: wazero.
			NewRuntimeConfigCompiler().
			WithCompilationCache(wazero.NewCompilationCache()).
			WithCloseOnContextDone(true),
	})
	if err != nil {
		return nil, fmt.Errorf("executor: %w", err)
	}

	server := host.Server{
		ViewProvider:   c,
		AnchorProvider: &anchor.Node{},
		PubSubProvider: &pubsub.Server{
			Log:         cfg.Logger,
			TopicJoiner: ps,
		},
		ExecutorProvider: e,
	}

	return &Server{
		closer: closer,
		Server: server,
		Router: c,
	}, nil
}

func (cfg Config) export(ctx context.Context, h local.Host, boot ClientProvider) {
	for _, proto := range proto.Namespace(cfg.NS) {
		h.SetStreamHandler(proto, func(s network.Stream) {
			defer s.Close()

			conn := rpc.NewConn(transport(s), &rpc.Options{
				ErrorReporter:   system.ErrorReporter{Logger: cfg.Logger},
				BootstrapClient: boot.Client(), // host.Host
			})
			defer conn.Close()

			select {
			case <-ctx.Done():
			case <-conn.Done():
			}
		})
	}
}

func transport(s network.Stream) rpc.Transport {
	if strings.HasSuffix(string(s.Protocol()), "/packed") {
		return rpc.NewPackedStreamTransport(s)
	}

	return rpc.NewStreamTransport(s)
}

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

type closeFunc func() error

func (close closeFunc) Close() error {
	return close()
}
