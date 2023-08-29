package ww

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/thejerf/suture/v4"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/wetware/pkg/cluster"
	"github.com/wetware/pkg/cluster/pulse"
	"github.com/wetware/pkg/cluster/routing"
	"github.com/wetware/pkg/util/proto"
)

func (vat Vat) ListenAndServe(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	app := suture.New(vat.String(), suture.Spec{
		EventHook: vat.OnEvent,
	})
	app.Add(vat)

	return app.Serve(ctx)
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
		conn, err := vat.Accept(ctx, nil) // FIXME:  export a bootstrap client
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

func (vat Vat) OnEvent(event suture.Event) {
	switch e := event.(type) {

	case suture.EventStopTimeout:
		slog.Error("shutdown failed",
			"event", e)

	case suture.EventServicePanic:
		slog.Error("crashed",
			"event", e)

	case suture.EventServiceTerminate:
		slog.Warn("terminated",
			"event", e)

	case *suture.EventBackoff:
		slog.Info("paused",
			"event", e)

	case suture.EventResume:
		slog.Info("resumed",
			"event", e)

	}
}
