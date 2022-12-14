package pubsub

import (
	"context"

	capnp "capnproto.org/go/capnp/v3"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"
	api "github.com/wetware/ww/internal/api/pubsub"
)

// Router is a client capability that confers the right to join pubsub
// topics.  It is the dual to Router.
type Router api.Router

// Join topic.  Callers MUST use the returned ReleaseFunc to leave the
// topic when finished.
func (ps Router) Join(ctx context.Context, topic string) (Topic, capnp.ReleaseFunc) {
	f, release := (api.Router)(ps).Join(ctx, func(ps api.Router_join_Params) error {
		return ps.SetName(topic)
	})

	return Topic(f.Topic()), release
}

func (ps Router) AddRef() Router {
	return Router(capnp.Client(ps).AddRef())
}

func (ps Router) Release() {
	capnp.Client(ps).Release()
}

/*
	Server
*/

// TopicJoiner can join libp2p pubsub topics.  It is a low-
// level interface provided to Router.
type TopicJoiner interface {
	Join(string, ...pubsub.TopicOpt) (*pubsub.Topic, error)
}

// Server for the pubsub capability.
type Server struct {
	Log         log.Logger
	TopicJoiner TopicJoiner
	topics      topicManager
}

func (r *Server) PubSub() Router {
	return NewJoiner(r)
}

func (r *Server) Client() capnp.Client {
	return capnp.Client(r.PubSub())
}

func (r *Server) Join(ctx context.Context, call MethodJoin) error {
	if r.Log == nil {
		r.Log = log.New()
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	name, err := call.Args().Name()
	if err != nil {
		return err
	}

	t, err := r.topics.GetOrCreate(r.Log, r.TopicJoiner, name)
	if err == nil {
		err = res.SetTopic(t)
	}

	return err
}
