package pubsub

import (
	"context"
	"sync"

	capnp "capnproto.org/go/capnp/v3"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	api "github.com/wetware/pkg/api/pubsub"
)

// topicManager is responsible for refcounting *pubsub.Topic instances.
type topicManager struct {
	mu     sync.Mutex
	topics map[string]*capnp.WeakClient
}

func (tm *topicManager) GetOrCreate(ctx context.Context, log Logger, ps TopicJoiner, name string) (api.Topic, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Fast path; we've already joined this topic.
	if t := tm.lookup(name); t != (api.Topic{}) {
		return t, nil
	}

	// Slow path; join the topic and add it to the map.

	return tm.join(log, ps, name)
}

// lookup an existing topic in the map.  Caller MUST hold mu.
func (tm *topicManager) lookup(name string) (t api.Topic) {
	if wc := tm.topics[name]; wc != nil {
		// The managedServer ensures we will never see a released
		// client in the map
		c, _ := wc.AddRef()
		t = api.Topic(c)
	}

	return
}

// join a topic and add it to the map.  Caller MUST hold mu.
func (tm *topicManager) join(log Logger, ps TopicJoiner, name string) (topic api.Topic, err error) {
	var t *pubsub.Topic
	if t, err = ps.Join(name); err == nil {
		// log = log.With("topic", name)
		topic = tm.asCapability(log, t)
	}

	return

}

// returns a capability for the supplied topic.  Caller MUST hold mu.
func (tm *topicManager) asCapability(log Logger, t *pubsub.Topic) api.Topic {
	if tm.topics == nil {
		tm.topics = make(map[string]*capnp.WeakClient)
	}

	topic := tm.newClient(log, t)
	tm.topics[t.String()] = capnp.Client(topic).WeakRef()

	return topic
}

func (tm *topicManager) newClient(log Logger, t *pubsub.Topic) api.Topic {
	server := &topicServer{
		log:   log,
		topic: t,
		leave: tm.leave,
	}

	hook := &managedServer{
		mu:         &tm.mu,
		ClientHook: api.Topic_NewServer(server),
	}

	return api.Topic(capnp.NewClient(hook))
}

func (tm *topicManager) leave(t *pubsub.Topic) error {
	delete(tm.topics, t.String())
	return t.Close()
}

// managedServer is a capnp.ClientHook that locks the topic manager
// during shutdown.
type managedServer struct {
	mu *sync.Mutex // topicManager.mu
	capnp.ClientHook
}

func (s *managedServer) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ClientHook.Shutdown()
}
