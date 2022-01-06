// package server exports the Wetware worker node.
package server

import (
	"io"
	"path"
	"strings"

	"capnproto.org/go/capnp/v3/rpc"
	"capnproto.org/go/capnp/v3/server"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/helpers"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"
	ctxutil "github.com/lthibault/util/ctx"

	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/ww/internal/api/client"
	rpcutil "github.com/wetware/ww/internal/util/rpc"
	ww "github.com/wetware/ww/pkg"
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
	ns  string
	log log.Logger

	h  host.Host
	ps PubSub
	c  *cluster.Node
}

func New(h host.Host, ps PubSub, opt ...Option) (n Node, err error) {
	n.id = uuid.Must(uuid.NewRandom()) // instance ID
	n.h = h
	n.ps = ps

	for _, option := range withDefaults(opt) {
		option(&n)
	}

	ctx := ctxutil.C(h.Network().Process().Closing())
	n.c, err = cluster.New(ctx, ps,
		cluster.WithLogger(n.log),
		cluster.WithNamespace(n.ns),
		/* cluster.WithMeta(...), */
		/* cluster.WithTTL(...), */) // TODO

	if err == nil {
		err = n.registerHandlers()
	}

	return
}

func (n Node) Close() error {
	n.unregisterHandlers()
	return nil
}

// String returns the cluster namespace
func (n Node) String() string { return n.ns }

func (n Node) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"ns":       n.ns,
		"id":       n.h.ID(),
		"instance": n.id,
	}
}

func (n Node) PubSub() PubSub { return n.ps }

func (n Node) handleRPC(f transportFactory) network.StreamHandler {
	return func(s network.Stream) {
		defer s.Close()

		h := client.Host_ServerToClient(n, &server.Policy{
			MaxConcurrentCalls: 64, // lower when capnp issue #190 is resolved
		})

		conn := rpc.NewConn(f.NewTransport(s), &rpc.Options{
			BootstrapClient: h.Client,
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

const (
	proto       = ww.Proto
	protoPacked = ww.Proto + "/packed"
)

func (n *Node) registerHandlers() error {
	matchVersion, err := helpers.MultistreamSemverMatcher(ww.Proto)
	if err != nil {
		return err
	}

	matchCluster := matchNamespace(n.ns).Then(matchVersion)

	n.h.SetStreamHandlerMatch(proto,
		matchCluster,
		n.handleRPC(rpc.NewStreamTransport))

	n.h.SetStreamHandlerMatch(protoPacked,
		matchPacked().Then(matchCluster),
		n.handleRPC(rpc.NewPackedStreamTransport))

	return nil
}

func (n *Node) unregisterHandlers() {
	n.h.RemoveStreamHandler(proto)
	n.h.RemoveStreamHandler(protoPacked)
}

type matcher func(string) bool

func (match matcher) Then(next matcher) matcher {
	return func(s string) (ok bool) {
		if ok = match(s); ok {
			ok = next(path.Dir(s)) // pop last element of path
		}
		return
	}
}

// /ww/<version>/<ns>[/packed]
func matchNamespace(ns string) matcher {
	return func(s string) bool {
		return path.Base(strings.TrimSuffix(s, "/packed")) == ns
	}
}

func matchPacked() matcher {
	return func(s string) bool {
		return path.Base(s) == "packed"
	}
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
