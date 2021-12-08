// package server exports the Wetware worker node.
package server

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"capnproto.org/go/capnp/v3/rpc"
	"capnproto.org/go/capnp/v3/server"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/helpers"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"

	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/ww/internal/api/client"
	rpcutil "github.com/wetware/ww/internal/util/rpc"
	ww "github.com/wetware/ww/pkg"
)

type Node struct {
	log log.Logger

	ns string
	ts []string // default topics

	host HostFactory
	dht  DHTFactory
	ps   PubSubFactory
}

func New(opt ...Option) (n Node) {
	for _, option := range withDefaults(opt) {
		option(&n)
	}

	return
}

// String returns the cluster namespace
func (n Node) String() string { return n.ns }

func (n Node) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"ns": n.ns,
	}
}

func (n Node) Serve(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if rh, ok := n.host.(RoutingHook); ok {
		rh.SetRouting(n.dht)
	}

	h, err := n.host.New(ctx)
	if err != nil {
		return fmt.Errorf("host: %w", err)
	}

	dht, err := n.dht.New(h)
	if err != nil {
		return fmt.Errorf("dht: %w", err)
	}

	ps, err := n.ps.New(h, dht)
	if err != nil {
		return fmt.Errorf("pubsub: %w", err)
	}

	c, err := cluster.New(ctx, ps, cluster.WithNamespace(n.ns))
	if err != nil {
		return err
	}

	return newInstance(n.log, n.ns, h, ps, c).Serve(ctx, n.ts)
}

type instance struct {
	id  uuid.UUID
	log log.Logger
	h   host.Host
	tm  *topicManager
	*cluster.Node
}

func newInstance(l log.Logger, ns string, h host.Host, ps PubSub, c *cluster.Node) instance {
	id := uuid.Must(uuid.NewRandom())
	l = l.With(log.F{
		"ns":       ns,
		"id":       h.ID(),
		"instance": id,
	})

	return instance{
		id:   id,
		log:  l,
		h:    h,
		tm:   newTopicManager(ps, c.Topic()),
		Node: c,
	}
}

func (in instance) Serve(ctx context.Context, topics []string) error {
	defer in.tm.Close()

	cancel, err := in.startRelays(topics)
	if err != nil {
		return fmt.Errorf("relay topics: %w", err)
	}
	defer cancel()

	if err := in.registerHandlers(in.Topic().String()); err != nil {
		return fmt.Errorf("register handlers: %w", err)
	}
	defer in.unregisterHandlers()

	in.log.WithField("addrs", in.h.Addrs()).Info("ready")
	<-ctx.Done()
	in.log.Warn("stopping")

	return in.Close()
}

func (in instance) startRelays(ts []string) (cancel func(), err error) {
	var cs = make([]pubsub.RelayCancelFunc, len(ts))

	cancel = func() {
		for _, cancel := range cs {
			cancel()
		}
	}

	for i, topic := range ts {
		if cs[i], err = in.tm.Relay(topic); err != nil {
			cs = cs[:i]
			cancel()
			break
		}

		in.log.WithField("topic", topic).Debug("relaying topic")
	}

	return
}

func (in instance) handleRPC(f transportFactory) network.StreamHandler {
	return func(s network.Stream) {
		defer s.Close()

		h := client.Host_ServerToClient(in, &server.Policy{
			MaxConcurrentCalls: 64, // lower when capnp issue #190 is resolved
		})

		conn := rpc.NewConn(f.NewTransport(s), &rpc.Options{
			BootstrapClient: h.Client,
			ErrorReporter: rpcutil.ErrReporterFunc(func(err error) {
				in.log.
					With(streamFields(s)).
					WithError(err).
					Debug("rpc error")
			}),
		})
		defer conn.Close()

		select {
		case <-conn.Done():
			in.log.
				With(streamFields(s)).
				Debug("client hung up")

		case <-in.h.Network().Process().Closing():
			in.log.
				With(streamFields(s)).
				Debug("server shutting down")
		}
	}
}

const (
	proto       = ww.Proto
	protoPacked = ww.Proto + "/packed"
)

func (p instance) registerHandlers(ns string) error {
	matchVersion, err := helpers.MultistreamSemverMatcher(ww.Proto)
	if err != nil {
		return err
	}

	matchCluster := matchNamespace(ns).Then(matchVersion)

	p.h.SetStreamHandlerMatch(proto,
		matchCluster,
		p.handleRPC(rpc.NewStreamTransport))

	p.h.SetStreamHandlerMatch(protoPacked,
		matchPacked().Then(matchCluster),
		p.handleRPC(rpc.NewPackedStreamTransport))

	return nil
}

func (p instance) unregisterHandlers() {
	p.h.RemoveStreamHandler(proto)
	p.h.RemoveStreamHandler(protoPacked)
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
