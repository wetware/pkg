package server

import (
	"context"
	"fmt"
	"strings"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/lthibault/log"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/anchor"
	host_cap "github.com/wetware/ww/pkg/host"
	"github.com/wetware/ww/pkg/pubsub"
)

type Config struct {
	Logger   log.Logger
	NS       string
	Join     []string // static bootstrap peers
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

func (cfg Config) Serve(ctx context.Context, h host.Host) error {
	if cfg.Logger == nil {
		cfg.Logger = log.New()
	}
	cfg.Logger = cfg.Logger.WithField("ns", cfg.NS)

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

	cfg.export(ctx, h, &host_cap.Server{
		ViewProvider:   c,
		AnchorProvider: &anchor.Node{},
		PubSubProvider: &pubsub.Server{
			Log:         cfg.Logger,
			TopicJoiner: ps,
		},
	})

	if err := c.Bootstrap(ctx); err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}

	<-ctx.Done()
	return ctx.Err()
}

func (cfg Config) export(ctx context.Context, h host.Host, s *host_cap.Server) {
	for _, proto := range []protocol.ID{
		ww.Subprotocol(cfg.NS),
		ww.Subprotocol(cfg.NS, "/packed"),
	} {
		h.SetStreamHandler(proto, cfg.handler(ctx, s))
	}
}

func (cfg Config) handler(ctx context.Context, h *host_cap.Server) network.StreamHandler {
	return func(s network.Stream) {
		defer s.Close()

		conn := rpc.NewConn(transport(s), &rpc.Options{
			BootstrapClient: h.Client(),
		})
		defer conn.Close()

		select {
		case <-ctx.Done():
		case <-conn.Done():
		}
	}
}

func transport(s network.Stream) rpc.Transport {
	if strings.HasSuffix(string(s.Protocol()), "/packed") {
		return rpc.NewPackedStreamTransport(s)
	}

	return rpc.NewStreamTransport(s)
}
