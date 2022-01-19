// package server exports the Wetware worker node.
package server

import (
	"io"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"
	ctxutil "github.com/lthibault/util/ctx"
	"go.uber.org/multierr"

	"github.com/wetware/casm/pkg/cluster"
	protoutil "github.com/wetware/casm/pkg/util/proto"
	rpcutil "github.com/wetware/ww/internal/util/rpc"
	ww "github.com/wetware/ww/pkg"
	pscap "github.com/wetware/ww/pkg/cap/pubsub"
)

type PubSub interface {
	Join(string, ...pubsub.TopicOpt) (*pubsub.Topic, error)
	RegisterTopicValidator(string, interface{}, ...pubsub.ValidatorOpt) error
	UnregisterTopicValidator(string) error
	GetTopics() []string
	ListPeers(topic string) []peer.ID
}

type Node struct {
	id  uuid.UUID
	log log.Logger

	h          host.Host
	ps         pscap.Factory
	clusterOpt []cluster.Option
	c          *cluster.Node
}

func New(h host.Host, ps PubSub, opt ...Option) (*Node, error) {
	ctx := ctxutil.C(h.Network().Process().Closing())

	var n = &Node{
		h:  h,
		id: uuid.Must(uuid.NewRandom()), // instance ID
		ps: pscap.New(ctx, ps),
	}

	for _, option := range withDefault(opt) {
		option(n)
	}

	// Start cluster
	var err error
	if n.c, err = cluster.New(ctx, ps, n.clusterOpt...); err != nil {
		return nil, err
	}

	n.registerHandlers()
	return n, n.c.Bootstrap(ctx)
}

func (n *Node) Close() error {
	n.h.RemoveStreamHandler(ww.Subprotocol(n.String()))
	n.h.RemoveStreamHandler(ww.Subprotocol(n.String(), "packed"))
	return multierr.Combine(
		n.c.Close(),
		n.ps.Close())
}

// String returns the cluster namespace
func (n *Node) String() string { return n.c.Topic().String() }

func (n *Node) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"ns":       n.String(),
		"id":       n.h.ID(),
		"instance": n.id,
	}
}

func (n *Node) handleRPC(f transportFactory) network.StreamHandler {
	return func(s network.Stream) {
		defer s.Close()

		conn := rpc.NewConn(f.NewTransport(s), &rpc.Options{
			BootstrapClient: n.ps.New(nil).Client,
			ErrorReporter: rpcutil.ErrReporterFunc(func(err error) {
				n.log.
					With(streamFields(s)).
					WithError(err).
					Debug("rpc error")
			}),
		})
		defer conn.Close()

		select {
		case <-conn.Done():
			n.log.
				With(streamFields(s)).
				Debug("client hung up")

		case <-n.h.Network().Process().Closing():
			n.log.
				With(streamFields(s)).
				Debug("server shutting down")
		}
	}
}

func (n *Node) registerHandlers() {
	var (
		match       = ww.NewMatcher(n.String())
		matchPacked = match.Then(protoutil.Exactly("packed"))
	)

	n.h.SetStreamHandlerMatch(
		ww.Subprotocol(n.String()),
		match,
		n.handleRPC(rpc.NewStreamTransport))

	n.h.SetStreamHandlerMatch(
		ww.Subprotocol(n.String(), "packed"),
		matchPacked,
		n.handleRPC(rpc.NewPackedStreamTransport))
}

type transportFactory func(io.ReadWriteCloser) rpc.Transport

func (f transportFactory) NewTransport(rwc io.ReadWriteCloser) rpc.Transport { return f(rwc) }

func streamFields(s network.Stream) log.F {
	return log.F{
		"peer":   s.Conn().RemotePeer(),
		"conn":   s.Conn().ID(),
		"proto":  s.Protocol(),
		"stream": s.ID(),
	}
}
