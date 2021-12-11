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
	"github.com/libp2p/go-eventbus"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/helpers"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"

	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/casm/pkg/util/service"
	"github.com/wetware/ww/internal/api/client"
	rpcutil "github.com/wetware/ww/internal/util/rpc"
	ww "github.com/wetware/ww/pkg"
)

type EvtNodeReady struct {
	ID        peer.ID
	Instance  uuid.UUID
	Namespace string
}

func (ev EvtNodeReady) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"id":       ev.ID,
		"instance": ev.Instance,
		"ns":       ev.Namespace,
	}
}

type Node struct {
	log log.Logger

	ts []string // default topics

	host HostFactory
	dht  DHTFactory
	ps   PubSubFactory
	cc   ClusterConfig
}

func New(opt ...Option) (n Node) {
	for _, option := range withDefaults(opt) {
		option(&n)
	}

	return
}

// String returns the cluster namespace
func (n Node) String() string { return "ww.cluster" }

func (n Node) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"ns":     n.cc.NS,
		"ttl":    n.cc.TTL,
		"topics": n.ts,
	}
}

func (n Node) Serve(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if rh, ok := n.host.(RoutingHook); ok {
		rh.SetRouting(n.dht)
	}

	h, err := n.host.New(ctx) // closed by instance
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

	return n.newInstance(h, ps).Serve(ctx, n.ts, n.cc.options())
}

type instance struct {
	id  uuid.UUID
	log log.Logger

	h  host.Host
	ps PubSub

	cc ClusterConfig
	ts []string
}

func (n Node) newInstance(h host.Host, ps PubSub) instance {
	id := uuid.Must(uuid.NewRandom())
	n.log = n.log.With(log.F{
		"addrs":    h.Addrs(),
		"ns":       n.cc.NS,
		"id":       h.ID(),
		"instance": id,
	})

	return instance{
		id:  id,
		log: n.log,
		h:   h,
		ps:  ps,
		ts:  n.ts,
		cc:  n.cc,
	}
}

func (in instance) Serve(ctx context.Context, topics []string, opt []cluster.Option) error {
	var (
		tm     *topicManager
		cancel func()
		c      *cluster.Node
		ss     = service.Set{
			// Capabilities
			service.Hook{
				OnStart: func() error {
					return in.registerHandlers(in.cc.NS)
				},
				OnClose: func() error {
					in.unregisterHandlers()
					return nil
				},
			},
			// Cluster
			service.Hook{
				OnStart: func() (err error) {
					c, err = cluster.New(ctx, in.ps,
						cluster.WithLogger(in.log),
						cluster.WithTTL(in.cc.TTL),
						cluster.WithMeta(in.cc.Meta),
						cluster.WithReadiness(in.cc.Ready),
						cluster.WithNamespace(in.cc.NS),
						cluster.WithRoutingTable(in.cc.Routing))
					return
				},
				OnClose: func() error {
					return c.Close()
				},
			},
			// Topic manager
			service.Hook{
				OnStart: func() (err error) {
					tm = newTopicManager(in.ps, c.Topic())
					cancel, err = in.startRelays(tm, topics)
					return
				},
				OnClose: func() error {
					cancel()
					return tm.Close()
				},
			},
			// Signalling
			service.Hook{
				OnStart: func() error {
					sub, err := in.h.EventBus().Subscribe(new(event.EvtLocalAddressesUpdated))
					if err != nil {
						return err
					}
					defer sub.Close()

					select {
					case <-sub.Out():
					case <-ctx.Done():
						return fmt.Errorf("wait host ready: %w", ctx.Err())
					}

					e, err := in.h.EventBus().Emitter(new(EvtNodeReady), eventbus.Stateful)
					if err != nil {
						return err
					}
					defer e.Close()

					return e.Emit(EvtNodeReady{
						ID:        in.h.ID(),
						Instance:  in.id,
						Namespace: in.cc.NS,
					})
				},
			}}
	)

	if err := ss.Start(); err != nil {
		return fmt.Errorf("setup: %w", err)
	}

	<-ctx.Done()

	if err := ss.Close(); err != nil {
		return fmt.Errorf("teardown")
	}

	return in.h.Close()
}

func (in instance) startRelays(tm *topicManager, ts []string) (cancel func(), err error) {
	var cs = make([]pubsub.RelayCancelFunc, len(ts))

	cancel = func() {
		for _, cancel := range cs {
			cancel()
		}
	}

	for i, topic := range ts {
		if cs[i], err = tm.Relay(topic); err != nil {
			cs = cs[:i]
			cancel()
			break
		}

		in.log.WithField("topic", topic).Info("relaying topic")
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
