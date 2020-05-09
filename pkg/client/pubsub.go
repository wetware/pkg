package client

import (
	"context"
	"strings"
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// Subscription to a topic.  C is automatically closed when the context
// passed to Topic.Subscribe expires.
type Subscription struct {
	C <-chan *pubsub.Message

	topic string
	sub   *pubsub.Subscription
}

type topicSet struct {
	ns string
	ps *pubsub.PubSub

	mu sync.Mutex
	ts map[string]Topic
}

func newTopicSet(ns string, ps *pubsub.PubSub) *topicSet {
	return &topicSet{
		ns: ns,
		ts: make(map[string]Topic),
		ps: ps,
	}
}

// Join a topic
func (s *topicSet) Join(topic string) (Topic, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := s.fullyQualifiedTopicName(topic)
	if t, ok := s.ts[name]; ok {
		return t, nil
	}

	top, err := s.ps.Join(name)
	if err != nil {
		return Topic{}, err
	}

	t := Topic{topic: name, t: top, mgr: topicManager{s}}
	s.ts[name] = t

	return t, nil
}

func (s *topicSet) fullyQualifiedTopicName(topic string) string {
	return strings.TrimRight(strings.Join([]string{s.ns, topic}, "."), ".")
}

// Topic handle
type Topic struct {
	topic string
	t     *pubsub.Topic

	mgr interface {
		RegisterValidator(string, pubsub.Validator, []pubsub.ValidatorOpt) error
		UnregisterValidator(string) error
		Clear(string)
	}
}

func (t Topic) String() string {
	return t.topic
}

// Close the topic.  Returns an error if there are active Subscriptions.
// Subsequent calls to Close return nil.
func (t Topic) Close() error {
	t.mgr.Clear(t.topic)
	return t.t.Close()
}

// Publish data to all topic subscribers
func (t Topic) Publish(ctx context.Context, b []byte) error {
	return t.t.Publish(ctx, b)
}

// Subscribe to the topic.  When the context passed to Subscribe expires, the
// returned subscription will be closed.
func (t Topic) Subscribe(ctx context.Context) (s Subscription, err error) {
	s.topic = t.topic
	if s.sub, err = t.t.Subscribe(); err != nil {
		return
	}

	ch := make(chan *pubsub.Message, 32)
	go func() {
		defer close(ch)

		for {
			msg, err := s.sub.Next(ctx)
			if err != nil {
				break
			}

			select {
			case ch <- msg:
			case <-ctx.Done():
				break
			}
		}
	}()

	s.C = ch
	return
}

// topicManager is a proxy to a topicSet for use by Topic.
type topicManager struct{ *topicSet }

func (mgr topicManager) RegisterValidator(topic string, v pubsub.Validator, opt []pubsub.ValidatorOpt) error {
	return mgr.ps.RegisterTopicValidator(topic, v, opt...)
}

func (mgr topicManager) UnregisterValidator(topic string) error {
	return mgr.ps.UnregisterTopicValidator(topic)
}

// NOTE: topic must be a fully-qualified topic name
func (mgr topicManager) Clear(topic string) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	delete(mgr.ts, topic)
}
