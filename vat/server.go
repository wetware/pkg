package vat

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	gossipsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/discovery"
	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/pkg/errors"

	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/boot"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/cluster"
	"github.com/wetware/pkg/cluster/pulse"
	"github.com/wetware/pkg/cluster/routing"
	"github.com/wetware/pkg/system"
	"github.com/wetware/pkg/util/proto"
)

var _ rpc.Network = (*server)(nil)

type Config struct {
	NS                 string
	Host               local.Host
	Bootstrap, Ambient discovery.Discovery
	Meta               pulse.Preparer
	Auth               auth.Policy
	OnJoin             func(auth.Session)
}

func (conf Config) Serve(ctx context.Context) error {
	return server{
		Config: conf,
		ch:     make(chan network.Stream),
	}.Serve(ctx)
}

func (conf Config) String() string {
	return fmt.Sprintf("%s:%s", conf.NS, conf.Host.ID())
}

func (conf Config) Logger() *slog.Logger {
	return slog.Default().With(
		"ns", conf.NS,
		"peer", conf.Host.ID())
}

type server struct {
	Config
	ch chan network.Stream
}

func (svr server) Serve(ctx context.Context) error {
	svr.ch = make(chan network.Stream)
	defer close(svr.ch)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ps, err := svr.NewPubSub(ctx)
	if err != nil {
		return err
	}

	rt := routing.New(time.Now())

	err = ps.RegisterTopicValidator(
		svr.NS,
		pulse.NewValidator(rt))
	if err != nil {
		return err
	}
	defer ps.UnregisterTopicValidator(svr.NS)

	t, err := ps.Join(svr.NS)
	if err != nil {
		return err
	}

	r := &cluster.Router{
		Topic:        t,
		Meta:         svr.Meta,
		RoutingTable: rt,
	}
	defer r.Close()

	server := &host.Server{
		Auth:         svr.Auth,
		ViewProvider: r,
		// PubSubProvider: &pubsub.Server{TopicJoiner: ps},
		// RuntimeConfig: wazero.NewRuntimeConfig().
		// 	WithCloseOnContextDone(true),
	}

	release, err := svr.Join(ctx, r, server)
	if err != nil {
		return err
	}
	defer release()

	logger := system.ErrorReporter{
		Logger: svr.Logger().With("id", r.ID()),
	}

	logger.Info("wetware started")
	defer logger.Warn("wetware started")

	for {
		opts := &rpc.Options{
			BootstrapClient: server.Export(),
			ErrorReporter:   logger,
		}

		conn, err := svr.Accept(ctx, opts)
		if err != nil {
			return err
		}

		remote := conn.RemotePeerID().Value.(peer.AddrInfo)
		logger.Info("accepted peer connection",
			"remote", remote.ID)
	}
}

func (svr server) NewPubSub(ctx context.Context) (*gossipsub.PubSub, error) {
	d := &boot.Namespace{
		Name:      svr.NS,
		Bootstrap: svr.Bootstrap,
		Ambient:   svr.Ambient,
	}

	return gossipsub.NewGossipSub(ctx, svr.Host,
		gossipsub.WithPeerExchange(true),
		// ps.WithRawTracer(svr.tracer()),
		gossipsub.WithDiscovery(d),
		gossipsub.WithProtocolMatchFn(protoMatchFunc(svr.NS)),
		gossipsub.WithGossipSubProtocols(subProtos(svr.NS)),
		gossipsub.WithPeerOutboundQueueSize(1024),
		gossipsub.WithValidateQueueSize(1024))
}

func (svr server) Join(ctx context.Context, r *cluster.Router, server *host.Server) (capnp.ReleaseFunc, error) {
	release := svr.bind(ctx) // registers stream handlers

	if err := r.Bootstrap(ctx); err != nil {
		defer release()
		return nil, err
	}

	return release, nil
}

func (svr server) bind(ctx context.Context) capnp.ReleaseFunc {
	for _, id := range proto.Namespace(svr.NS) {
		svr.Host.SetStreamHandler(id, svr.handler(ctx))
	}

	return func() {
		for _, id := range proto.Namespace(svr.NS) {
			svr.Host.RemoveStreamHandler(id)
		}
	}
}

func (svr server) handler(ctx context.Context) network.StreamHandler {
	return func(s network.Stream) {
		select {
		case svr.ch <- s:
		case <-ctx.Done():
		}
	}
}

// Return the identifier for caller on this network.
func (svr server) LocalID() rpc.PeerID {
	return rpc.PeerID{
		Value: svr.Host.ID(),
	}
}

// Connect to another peer by ID. The supplied Options are used
// for the connection, with the values for RemotePeerID and Network
// overridden by the Network.
func (svr server) Dial(pid rpc.PeerID, opt *rpc.Options) (*rpc.Conn, error) {
	ctx := context.TODO()

	opt.RemotePeerID = pid
	opt.Network = svr

	peer := pid.Value.(peer.AddrInfo)
	protos := proto.Namespace(svr.NS)

	s, err := svr.Host.NewStream(ctx, peer.ID, protos...)
	if err != nil {
		return nil, err
	}

	conn := rpc.NewConn(transport(s), opt)
	return conn, nil
}

// Accept the next incoming connection on the network, using the
// supplied Options for the connection. Generally, callers will
// want to invoke this in a loop when launching a server.
func (svr server) Accept(ctx context.Context, opt *rpc.Options) (*rpc.Conn, error) {
	select {
	case s, ok := <-svr.ch:
		if !ok {
			return nil, errors.New("server closed")
		}

		opt.RemotePeerID.Value = peer.AddrInfo{
			ID:    s.Conn().RemotePeer(),
			Addrs: svr.Host.Peerstore().Addrs(s.Conn().RemotePeer()),
		}
		opt.Network = svr

		conn := rpc.NewConn(transport(s), opt)
		return conn, nil

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Introduce the two connections, in preparation for a third party
// handoff. Afterwards, a Provide messsage should be sent to
// provider, and a ThirdPartyCapId should be sent to recipient.
func (svr server) Introduce(provider, recipient *rpc.Conn) (rpc.IntroductionInfo, error) {
	return rpc.IntroductionInfo{}, errors.New("NOT IMPLEMENTED")
}

// Given a ThirdPartyCapID, received from introducedBy, connect
// to the third party. The caller should then send an Accept
// message over the returned Connection.
func (svr server) DialIntroduced(capID rpc.ThirdPartyCapID, introducedBy *rpc.Conn) (*rpc.Conn, rpc.ProvisionID, error) {
	return nil, rpc.ProvisionID{}, errors.New("NOT IMPLEMENTED")
}

// Given a RecipientID received in a Provide message via
// introducedBy, wait for the recipient to connect, and
// return the connection formed. If there is already an
// established connection to the relevant Peer, this
// SHOULD return the existing connection immediately.
func (svr server) AcceptIntroduced(recipientID rpc.RecipientID, introducedBy *rpc.Conn) (*rpc.Conn, error) {
	return nil, errors.New("NOT IMPLEMENTED")
}

func protoMatchFunc(ns string) gossipsub.ProtocolMatchFn {
	match := matcher(ns)

	return func(local protocol.ID) func(protocol.ID) bool {
		if match.Match(local) {
			return match.Match
		}

		panic(fmt.Sprintf("match failed for local protocol %s", local))
	}
}

func matcher(ns string) proto.MatchFunc {
	base, version := proto.Split(gossipsub.GossipSubID_v11)
	return proto.Match(
		proto.NewMatcher(ns),
		proto.Exactly(string(base)),
		proto.SemVer(string(version)))
}

func subProtos(ns string) ([]protocol.ID, func(gossipsub.GossipSubFeature, protocol.ID) bool) {
	return []protocol.ID{protoID(ns)}, features(ns)
}

// /ww/<version>/<ns>/meshsub/1.1.0
func protoID(ns string) protocol.ID {
	return proto.Join(
		proto.Root(ns),
		gossipsub.GossipSubID_v11)
}

func features(ns string) func(gossipsub.GossipSubFeature, protocol.ID) bool {
	supportGossip := matcher(ns)

	_, version := proto.Split(protoID(ns))
	supportsPX := proto.Suffix(version)

	return func(feat gossipsub.GossipSubFeature, proto protocol.ID) bool {
		switch feat {
		case gossipsub.GossipSubFeatureMesh:
			return supportGossip.Match(proto)

		case gossipsub.GossipSubFeaturePX:
			return supportsPX.Match(proto)

		default:
			return false
		}
	}
}
