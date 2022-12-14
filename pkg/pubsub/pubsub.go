//go:generate mockgen -source=pubsub.go -destination=../../internal/mock/pkg/pubsub/pubsub.go -package=mock_pubsub

package pubsub

import (
	"context"

	api "github.com/wetware/ww/internal/api/pubsub"
)

type (
	// MethodJoin is the server-side method parameter for JoinServer's
	// Join method.  See:  NewJoiner.
	MethodJoin = api.Router_join

	// MethodName is the server-side method parameter for TopicServer's
	// Join method.  See:  NewJoiner.
	MethodName = api.Topic_name

	// MethodPublish is the server-side method parameter for TopicServer's
	// Join method.  See:  NewJoiner.
	MethodPublish = api.Topic_publish

	// MethodSubscribe is the server-side method parameter for TopicServer's
	// Join method.  See:  NewJoiner.
	MethodSubscribe = api.Topic_subscribe
)

// JoinServer is an interface that allows alternate server implementations
// for Joiner.  The default implementation is Router.  See: NewJoiner.
type JoinServer interface {
	Join(context.Context, MethodJoin) error
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
	Name(context.Context, MethodName) error
	Publish(context.Context, MethodPublish) error
	Subscribe(context.Context, MethodSubscribe) error
}

// NewTopic returns a Joiner (a capability client) from a JoinServer
// interface.  This is most commonly used in unit-testing.
func NewTopic(s TopicServer) Topic {
	return Topic(api.Topic_ServerToClient(s))
}
