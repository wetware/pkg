package server

import (
	"fmt"
	"io"
	"sync"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	disc "github.com/libp2p/go-libp2p-discovery"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ctxutil "github.com/lthibault/util/ctx"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"
)

type PubSub interface {
	Join(topic string, opt ...pubsub.TopicOpt) (*pubsub.Topic, error)
	Subscribe(topic string, opts ...pubsub.SubOpt) (*pubsub.Subscription, error)
	GetTopics() []string
	ListPeers(topic string) []peer.ID
	BlacklistPeer(pid peer.ID)
	RegisterTopicValidator(topic string, val interface{}, opts ...pubsub.ValidatorOpt) error
	UnregisterTopicValidator(topic string) error
}

type PubSubFactory interface {
	New(host.Host, routing.ContentRouting) (PubSub, error)
}

type GossipsubFactory struct {
	fx.In

	// Bootstrap discovery.  These will be wrapped in a peer-sampling
	// cache and used to bootstrap the cluster.
	Advertiser discovery.Advertiser
	Discoverer discovery.Discoverer
}

func (GossipsubFactory) New(h host.Host, r routing.ContentRouting) (PubSub, error) {
	ctx := ctxutil.C(h.Network().Process().Closing())
	return pubsub.NewGossipSub(ctx, h,
		pubsub.WithDiscovery(disc.NewRoutingDiscovery(r)))
}

type topicManager struct {
	ps      PubSub
	cluster *pubsub.Topic

	once sync.Once
	wg   sync.WaitGroup
	mu   sync.Mutex
	ts   map[string]*pubsub.Topic
}

// newTopicManager maintains a set of active relays.
// The 'c' argument is the cluster topic, and SHOULD NOT
// be exported to users.
func newTopicManager(ps PubSub, c *pubsub.Topic) *topicManager {
	return &topicManager{
		ps:      ps,
		cluster: c,
	}
}

// Relay MUST NOT be called after the 'instance' has begun shutting down.
func (tm *topicManager) Relay(topic string) (cancel pubsub.RelayCancelFunc, err error) {
	tm.once.Do(func() {
		tm.ts = make(map[string]*pubsub.Topic)
	})

	// scope the topic to the cluster namespace
	topic = fmt.Sprintf("%s.%s",
		tm.cluster.String(),
		topic)

	tm.mu.Lock()
	t, ok := tm.ts[topic]
	tm.mu.Unlock()

	if !ok {
		if t, err = tm.ps.Join(topic); err != nil {
			return nil, fmt.Errorf("join '%s': %w", topic, err)
		}
	}

	if cancel, err = t.Relay(); err != nil {
		return nil, fmt.Errorf("relay '%s': %w", topic, err)
	}

	tm.wg.Add(1) // track additional relay
	cancel = func() { tm.wg.Done() }

	// TODO:  close 't' and remove from 'tm.ts' when all relays
	//        have been canceled.

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// topic may have been added in the meantime
	if _, ok = tm.ts[topic]; !ok {
		tm.ts[topic] = t
	}

	return
}

func (tm *topicManager) Close() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.wg.Wait() // wait for all relays to be canceled

	// NOTE:  'tm.cluster' is closed by the CASM 'cluster.Node'.
	//         Likewise, the pubsub's lifecycle is handled independently.
	//
	//         DO NOT close either of these objects.
	var g errgroup.Group
	for topic, t := range tm.ts {
		g.Go(closer(topic, t))
	}
	return g.Wait()
}

func closer(topic string, c io.Closer) func() error {
	return func() (err error) {
		if err = c.Close(); err != nil {
			err = fmt.Errorf("close '%s': %w", topic, err)
		}
		return
	}
}
