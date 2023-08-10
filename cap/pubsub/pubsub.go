//go:generate mockgen -source=pubsub.go -destination=test/pubsub.go -package=test_pubsub

package pubsub

import (
	"context"

	api "github.com/wetware/pkg/api/pubsub"
)

// JoinServer is an interface that allows alternate server implementations
// for Joiner.  The default implementation is Router.  See: NewJoiner.
type JoinServer interface {
	Join(context.Context, api.Router_join) error
}

// NewJoiner returns a Joiner (a capability client) from a JoinServer
// interface. Most users will prefer to instantiate a Joiner directly
// by calling the Router.PubSub method.
//
// NewJoiner allows callers to supply alternate implementations of
// JoinServer.  This is most commonly used in unit-testing.
func NewJoiner(s JoinServer) Router {
	return Router(api.Router_ServerToClient(s))
}

// TopicServer is an interface that allows alternate server implementations
// for Topic.  The default implementation is unexported.  See NewTopic.
type TopicServer interface {
	Name(context.Context, api.Topic_name) error
	Publish(context.Context, api.Topic_publish) error
	Subscribe(context.Context, api.Topic_subscribe) error
}

// NewTopic returns a Joiner (a capability client) from a JoinServer
// interface.  This is most commonly used in unit-testing.
func NewTopic(s TopicServer) Topic {
	return Topic(api.Topic_ServerToClient(s))
}
