package server

import (
	"context"
	"fmt"
	"time"

	p2p "github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"golang.org/x/exp/slog"

	"github.com/wetware/pkg/cluster"
	"github.com/wetware/pkg/cluster/pulse"
	"github.com/wetware/pkg/cluster/routing"
	"github.com/wetware/pkg/util/proto"
)

func DefaultP2POpts(opt ...p2p.Option) []p2p.Option {
	return append([]p2p.Option{
		p2p.NoTransports,
		p2p.Transport(tcp.NewTCPTransport),
		p2p.Transport(quic.NewTransport),
	}, opt...)
}

func ListenP2P(listen ...string) (local.Host, error) {
	return p2p.New(DefaultP2POpts(p2p.ListenAddrStrings(listen...))...)
}

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

	// FIXME:  register libp2p stream handlers here

	if err = r.Bootstrap(ctx); err != nil {
		return err
	}

	for {
		// XXX:  pass in a non-nil signer.  A nil signer
		// safely defaults to capnp.Client{}
		conn, err := vat.Accept(ctx, nil)
		if err != nil {
			return err
		}

		remote := conn.RemotePeerID().Value.(peer.ID)
		slog.Info("accepted peer connection",
			"remote", remote)
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
