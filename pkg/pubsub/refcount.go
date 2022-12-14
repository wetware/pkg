package pubsub

import (
	"context"
	"sync"

	capnp "capnproto.org/go/capnp/v3"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"
	api "github.com/wetware/ww/internal/api/pubsub"
)

// topicManager is responsible for refcounting *pubsub.Topic instances.
type topicManager struct {
	mu     sync.Mutex
	topics map[string]*managedServer
}

func (tm *topicManager) GetOrCreate(ctx context.Context, log log.Logger, ps TopicJoiner, name string) (api.Topic, error) {
	log = log.WithField("topic", name)

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// do we have one, already?
	if server, ok := tm.topics[name]; ok {
		defer log.Trace("topic ref acquired")
		return server.NewClient(), nil
	}

	// slow path...

	return tm.join(log, ps, name)
}

// join a topic and add it to the map.  Caller MUST hold mu.
func (tm *topicManager) join(log log.Logger, ps TopicJoiner, name string) (topic api.Topic, err error) {
	defer log.Debug("joined topic")

	var t *pubsub.Topic
	if t, err = ps.Join(name); err == nil {
		topic = tm.asCapability(log, t)
	}

	return

}

// returns a capability for the supplied topic.  Caller MUST hold mu.
func (tm *topicManager) asCapability(log log.Logger, t *pubsub.Topic) api.Topic {
	if tm.topics == nil {
		tm.topics = make(map[string]*managedServer)
	}

	server := tm.newTopicServer(log, t)
	tm.topics[t.String()] = server

	return server.NewClient()
}

func (tm *topicManager) newTopicServer(log log.Logger, t *pubsub.Topic) *managedServer {
	server := &topicServer{
		log:   log,
		topic: t,
		leave: tm.leave,
	}

	return &managedServer{
		ClientHook: api.Topic_NewServer(server),
		mu:         &tm.mu,
	}
}

func (tm *topicManager) leave(t *pubsub.Topic) error {
	delete(tm.topics, t.String())
	return t.Close()
}

// managedServer is a capnp.ClientHook that locks the topic manager
// during shutdown.
type managedServer struct {
	mu   *sync.Mutex // topicManager.mu
	refs int
	capnp.ClientHook
}

// NewClient returns an api.Topic and increments the refcount.
// Callers MUST hold mu.
func (s *managedServer) NewClient() api.Topic {
	s.refs++

	return api.Topic(capnp.NewClient(s))
}

func (s *managedServer) Shutdown() {
	// Prevent concurrent goroutines from manipulating the topic map
	// during shutdown.
	s.mu.Lock()
	defer s.mu.Unlock()

	// decrement the refcount and check that it's valid
	if s.refs--; s.refs < 0 {
		panic("refcounting error:  released server with zero refs")
	}

	if s.refs == 0 {
		s.ClientHook.Shutdown()
	}
}
