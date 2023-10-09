package vat

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	gossipsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/discovery"
	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/host/eventbus"
	"github.com/pkg/errors"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"go.uber.org/multierr"
	"zenhack.net/go/util/rc"

	api "github.com/wetware/pkg/api/cluster"
	"github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/boot"
	csp_server "github.com/wetware/pkg/cap/csp/server"
	"github.com/wetware/pkg/cluster"
	"github.com/wetware/pkg/cluster/pulse"
	"github.com/wetware/pkg/cluster/routing"
	"github.com/wetware/pkg/system"
	"github.com/wetware/pkg/util/proto"
)

var _ rpc.Network = (*Server)(nil)

type Config struct {
	NS                 string
	Host               local.Host
	Bootstrap, Ambient discovery.Discovery
	Meta               pulse.Preparer
	Auth               auth.Policy
	RuntimeConfig      wazero.RuntimeConfig
}

func (conf Config) Serve(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// We will use the libp2p Host's event bus to signal asynchronously
	// to other threads in the same process.  Applications wishing to
	// embed a wetware server can use this interface to respond to events
	// in the local vat.
	bus := conf.Host.EventBus()

	// We will signal changes to the server's root state.  One such signal
	// is produced for each call to Config.Serve().  The idea is that this
	// can later be wrapped in a supervision tree may call Serve repeatedly.
	// Applications should model this behavior as a stream of auth.Session
	// instances.
	state, err := bus.Emitter(new(auth.Session),
		eventbus.Stateful)
	if err != nil {
		return err
	}
	defer state.Close()

	ps, err := conf.NewPubSub(ctx)
	if err != nil {
		return err
	}

	rt := routing.New(time.Now())

	err = ps.RegisterTopicValidator(
		conf.NS,
		pulse.NewValidator(rt))
	if err != nil {
		return err
	}
	defer ps.UnregisterTopicValidator(conf.NS)

	t, err := ps.Join(conf.NS)
	if err != nil {
		return err
	}

	r := &cluster.Router{
		Topic:        t,
		Meta:         conf.Meta,
		RoutingTable: rt,
	}
	defer r.Close()

	root, err := conf.NewRootSession(r)
	if err != nil {
		return err
	}
	defer root.Logout()

	// e, err := conf.NewExecutor(ctx)
	// if err != nil {
	// 	return err
	// }

	server := &Server{
		NS:      conf.NS,
		Host:    conf.Host,
		Auth:    conf.Auth,
		Root:    root,
		Cluster: r,
		OnJoin:  state,
	}
	defer server.Close()

	release, err := server.Join(ctx, r)
	if err != nil {
		return err
	}
	defer release()

	logger := conf.Logger().With("id", r.ID())

	logger.Info("wetware started")
	defer logger.Warn("wetware stopped")

	for {
		opt := &rpc.Options{
			BootstrapClient: server.Export(),
			ErrorReporter:   system.ErrorReporter{Logger: logger},
		}

		conn, err := server.Accept(ctx, opt)
		if err != nil {
			return err
		}

		remote := conn.RemotePeerID().Value.(peer.AddrInfo)
		logger.Info("accepted peer connection",
			"remote", remote.ID)
	}
}

func (conf Config) NewRootSession(r *cluster.Router) (auth.Session, error) {
	_, seg := capnp.NewSingleSegmentMessage(nil)

	sess, err := core.NewRootSession(seg) // TODO(optimization):  non-root?
	if err != nil {
		return auth.Session{}, err
	}

	routingID := uint64(r.ID())
	sess.Local().SetServer(routingID)

	hostname, err := sess.Local().Host()
	if err != nil {
		return auth.Session{}, err
	}

	// Write session data
	err = multierr.Combine(
		sess.Local().SetHost(hostname),
		sess.Local().SetPeer(string(conf.Host.ID())),
		sess.SetView(api.View(r.View())),
	)

	return auth.Session(sess), err
}

func (conf Config) NewExecutor(ctx context.Context) (csp_server.Runtime, error) {
	if conf.RuntimeConfig == nil {
		if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
			conf.RuntimeConfig = wazero.
				NewRuntimeConfigCompiler().
				WithCompilationCache(wazero.NewCompilationCache()).
				WithCloseOnContextDone(true)
		} else {
			conf.RuntimeConfig = wazero.
				NewRuntimeConfigInterpreter().
				WithCompilationCache(wazero.NewCompilationCache()).
				WithCloseOnContextDone(true)
		}
	}

	r := wazero.NewRuntimeWithConfig(ctx, conf.RuntimeConfig)
	_, err := wasi_snapshot_preview1.Instantiate(ctx, r)
	if err != nil {
		return csp_server.Runtime{}, err
	}

	return csp_server.Runtime{
		Runtime: r,
		Cache:   make(csp_server.BytecodeCache),
		Tree:    csp_server.NewProcTree(ctx),
		Log:     slog.Default(),
	}, nil
}

func (conf Config) String() string {
	return fmt.Sprintf("%s:%s", conf.NS, conf.Host.ID())
}

func (conf Config) Logger() *slog.Logger {
	return slog.Default().With(
		"ns", conf.NS,
		"peer", conf.Host.ID())
}

func (conf Config) NewPubSub(ctx context.Context) (*gossipsub.PubSub, error) {
	d := &boot.Namespace{
		Name:      conf.NS,
		Bootstrap: conf.Bootstrap,
		Ambient:   conf.Ambient,
	}

	return gossipsub.NewGossipSub(ctx, conf.Host,
		gossipsub.WithPeerExchange(true),
		// ps.WithRawTracer(svr.tracer()),
		gossipsub.WithDiscovery(d),
		gossipsub.WithProtocolMatchFn(protoMatchFunc(conf.NS)),
		gossipsub.WithGossipSubProtocols(subProtos(conf.NS)),
		gossipsub.WithPeerOutboundQueueSize(1024),
		gossipsub.WithValidateQueueSize(1024))
}

func (svr *Server) Join(ctx context.Context, r *cluster.Router) (capnp.ReleaseFunc, error) {
	ref := svr.bindHandlers(ctx) // registers stream handlers
	defer ref.Release()

	// Send our first heartbeat to the cluster
	// topic, and kick off background tasks.
	if err := svr.OnJoin.Emit(svr.Root); err != nil {
		return nil, err
	} else if err = r.Bootstrap(ctx); err != nil {
		return nil, err
	}

	// Increment the refcount and return the new function's releaser.
	// Releasing the ref will cause libp2p stream handlers to be removed.
	release := ref.AddRef().Release
	return release, nil
}

func (svr *Server) bindHandlers(ctx context.Context) *rc.Ref[any] {
	for _, id := range proto.Namespace(svr.NS) {
		svr.Host.SetStreamHandler(id, svr.handler(ctx))
	}

	return rc.NewRef(any(nil), func() {
		for _, id := range proto.Namespace(svr.NS) {
			svr.Host.RemoveStreamHandler(id)
		}
	})
}

func (svr *Server) handler(ctx context.Context) network.StreamHandler {
	return func(s network.Stream) {
		select {
		case svr.ch <- s:
		case <-ctx.Done():
		}
	}
}

// Return the identifier for caller on this network.
func (svr *Server) LocalID() rpc.PeerID {
	return rpc.PeerID{
		Value: svr.Host.ID(),
	}
}

// Connect to another peer by ID. The supplied Options are used
// for the connection, with the values for RemotePeerID and Network
// overridden by the Network.
func (svr *Server) Dial(pid rpc.PeerID, opt *rpc.Options) (*rpc.Conn, error) {
	svr.setup()
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
func (svr *Server) Accept(ctx context.Context, opt *rpc.Options) (*rpc.Conn, error) {
	svr.setup()

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
		go func() {
			select {
			case <-conn.Done():
			case <-ctx.Done():
				conn.Close()
			}
		}()

		return conn, nil

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Introduce the two connections, in preparation for a third party
// handoff. Afterwards, a Provide messsage should be sent to
// provider, and a ThirdPartyCapId should be sent to recipient.
func (svr *Server) Introduce(provider, recipient *rpc.Conn) (rpc.IntroductionInfo, error) {
	return rpc.IntroductionInfo{}, errors.New("NOT IMPLEMENTED")
}

// Given a ThirdPartyCapID, received from introducedBy, connect
// to the third party. The caller should then send an Accept
// message over the returned Connection.
func (svr *Server) DialIntroduced(capID rpc.ThirdPartyCapID, introducedBy *rpc.Conn) (*rpc.Conn, rpc.ProvisionID, error) {
	return nil, rpc.ProvisionID{}, errors.New("NOT IMPLEMENTED")
}

// Given a RecipientID received in a Provide message via
// introducedBy, wait for the recipient to connect, and
// return the connection formed. If there is already an
// established connection to the relevant Peer, this
// SHOULD return the existing connection immediately.
func (svr *Server) AcceptIntroduced(recipientID rpc.RecipientID, introducedBy *rpc.Conn) (*rpc.Conn, error) {
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
