package pubsub

import (
	"context"
	"sync"

	capnp "capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/server"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"
	api "github.com/wetware/ww/internal/api/pubsub"
)

// topicManager is responsible for refcounting *pubsub.Topic instances.
type topicManager struct {
	mu     sync.Mutex
	topics map[string]api.Topic
}

func (tm *topicManager) GetOrCreate(ctx context.Context, log log.Logger, ps TopicJoiner, name string) (api.Topic, error) {
	log = log.WithField("topic", name)

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// do we have one, already?
	if t := tm.topics[name]; capnp.Client(t).IsValid() {
		defer log.Trace("topic ref acquired")
		return t.AddRef(), nil
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
		tm.topics = make(map[string]api.Topic)
	}

	topic := tm.newTopic(log, t)
	tm.topics[t.String()] = topic

	return topic
}

func (tm *topicManager) newTopic(log log.Logger, t *pubsub.Topic) api.Topic {
	server := &topicServer{
		log:   log,
		topic: t,
		leave: tm.leave,
	}

	hook := &managedServer{
		Server: api.Topic_NewServer(server),
		mu:     &tm.mu,
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
	*server.Server
}

func (s managedServer) Shutdown() {
	// Prevent concurrent goroutines from manipulating the topic map
	// during shutdown.
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Server.Shutdown()
}
