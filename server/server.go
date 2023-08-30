package server

import (
	"context"
	"fmt"
	"time"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/tetratelabs/wazero"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/cluster"
	"github.com/wetware/pkg/cluster/pulse"
	"github.com/wetware/pkg/cluster/routing"
	"github.com/wetware/pkg/system"
	"github.com/wetware/pkg/util/proto"
	"golang.org/x/exp/slog"
)

func (vat Vat) Serve(ctx context.Context) error {
	vat.ch = make(chan network.Stream)
	defer close(vat.ch)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	pubsub, err := pubsub.NewGossipSub(ctx, vat.Host,
		pubsub.WithPeerExchange(true),
		// pubsub.WithRawTracer(vat.tracer()),
		pubsub.WithDiscovery(vat.NS),
		pubsub.WithProtocolMatchFn(protoMatchFunc(vat.NS.Name)),
		pubsub.WithGossipSubProtocols(subProtos(vat.NS.Name)),
		pubsub.WithPeerOutboundQueueSize(1024),
		pubsub.WithValidateQueueSize(1024))
	if err != nil {
		return err
	}

	rt := routing.New(time.Now())

	err = pubsub.RegisterTopicValidator(
		vat.NS.Name,
		pulse.NewValidator(rt))
	if err != nil {
		return err
	}
	defer pubsub.UnregisterTopicValidator(vat.NS.Name)

	t, err := pubsub.Join(vat.NS.Name)
	if err != nil {
		return err
	}

	r := &cluster.Router{
		Topic:        t,
		Meta:         vat.Meta,
		RoutingTable: rt,
	}
	defer r.Close()

	// register stream handlers
	release := vat.bind(ctx)
	defer release()

	// join the cluster
	if err = r.Bootstrap(ctx); err != nil {
		return err
	}

	logger := vat.Logger().With("id", r.ID())
	logger.Info("wetware started")
	defer logger.Warn("wetware started")

	server := host.Server{
		ViewProvider: r,
		TopicJoiner:  pubsub,
		RuntimeConfig: wazero.
			NewRuntimeConfigCompiler().
			WithCloseOnContextDone(true),
	}

	host := server.Host()
	defer host.Release()

	for {
		conn, err := vat.Accept(ctx, &rpc.Options{
			BootstrapClient: capnp.Client(host.AddRef()),
			ErrorReporter: system.ErrorReporter{
				Logger: logger,
			},
		})
		if err != nil {
			return err
		}

		remote := conn.RemotePeerID().Value.(peer.AddrInfo)
		slog.Info("accepted peer connection",
			"remote", remote.ID)
	}
}

func (vat Vat) bind(ctx context.Context) capnp.ReleaseFunc {
	for _, id := range vat.NS.Protocols() {
		vat.Host.SetStreamHandler(id, vat.handler(ctx))
	}

	return func() {
		for _, id := range vat.NS.Protocols() {
			vat.Host.RemoveStreamHandler(id)
		}
	}
}

func (vat Vat) handler(ctx context.Context) network.StreamHandler {
	return func(s network.Stream) {
		select {
		case vat.ch <- s:
		case <-ctx.Done():
		}
	}
}

func protoMatchFunc(ns string) pubsub.ProtocolMatchFn {
	match := matcher(ns)

	return func(local protocol.ID) func(protocol.ID) bool {
		if match.Match(local) {
			return match.Match
		}

		panic(fmt.Sprintf("match failed for local protocol %s", local))
	}
}

func matcher(ns string) proto.MatchFunc {
	base, version := proto.Split(pubsub.GossipSubID_v11)
	return proto.Match(
		proto.NewMatcher(ns),
		proto.Exactly(string(base)),
		proto.SemVer(string(version)))
}

func subProtos(ns string) ([]protocol.ID, func(pubsub.GossipSubFeature, protocol.ID) bool) {
	return []protocol.ID{protoID(ns)}, features(ns)
}

// /ww/<version>/<ns>/meshsub/1.1.0
func protoID(ns string) protocol.ID {
	return proto.Join(
		proto.Root(ns),
		pubsub.GossipSubID_v11)
}

func features(ns string) func(pubsub.GossipSubFeature, protocol.ID) bool {
	supportGossip := matcher(ns)

	_, version := proto.Split(protoID(ns))
	supportsPX := proto.Suffix(version)

	return func(feat pubsub.GossipSubFeature, proto protocol.ID) bool {
		switch feat {
		case pubsub.GossipSubFeatureMesh:
			return supportGossip.Match(proto)

		case pubsub.GossipSubFeaturePX:
			return supportsPX.Match(proto)

		default:
			return false
		}
	}
}
