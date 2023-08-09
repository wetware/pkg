package pubsub_test

// import (
// 	"context"
// 	"crypto/rand"
// 	"fmt"
// 	"testing"

// 	"capnproto.org/go/capnp/v3/rpc"
// 	"capnproto.org/go/capnp/v3/rpc/transport"
// 	"github.com/libp2p/go-libp2p"
// 	pubsub "github.com/libp2p/go-libp2p-pubsub"
// 	"github.com/libp2p/go-libp2p/core/crypto"
// 	"github.com/libp2p/go-libp2p/core/protocol"
// 	inproc "github.com/lthibault/go-libp2p-inproc-transport"
// 	"github.com/stretchr/testify/require"
// 	protoutil "github.com/wetware/casm/pkg/util/proto"
// 	"github.com/wetware/ww"
// 	pscap "github.com/wetware/ww/pkg/pubsub"
// 	"golang.org/x/sync/errgroup"
// )

// const (
// 	ns      = "ns"
// 	topic   = "benchmark"
// 	payload = "benchmark payload"
// )

// var (
// 	h, _ = libp2p.New(
// 		libp2p.NoTransports,
// 		libp2p.Transport(inproc.New()),
// 	)
// )

// func BenchmarkPubSub(b *testing.B) {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	ps, err := pubsub.NewGossipSub(ctx, h,
// 		pubsub.WithPeerExchange(true),
// 		pubsub.WithProtocolMatchFn(ProtoMatchFunc()),
// 		pubsub.WithGossipSubProtocols(Subprotocols()))
// 	require.NoError(b, err)

// 	topic, err := ps.Join(topic)
// 	require.NoError(b, err)

// 	group, ctx := errgroup.WithContext(ctx)

// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		group.Go(func() error { return topic.Publish(ctx, []byte(payload)) })
// 	}

// 	require.NoError(b, group.Wait())
// }

// func BenchmarkPubSubNosign(b *testing.B) {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	ps, err := pubsub.NewGossipSub(ctx, h,
// 		pubsub.WithPeerExchange(true),
// 		pubsub.WithProtocolMatchFn(ProtoMatchFunc()),
// 		pubsub.WithGossipSubProtocols(Subprotocols()),
// 		pubsub.WithMessageSignaturePolicy(pubsub.LaxNoSign))
// 	require.NoError(b, err)

// 	topic, err := ps.Join(topic)
// 	require.NoError(b, err)

// 	group, ctx := errgroup.WithContext(ctx)

// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		group.Go(func() error { return topic.Publish(ctx, []byte(payload)) })
// 	}

// 	require.NoError(b, group.Wait())
// }

// func BenchmarkPubSubEd25519(b *testing.B) {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	priv, err := randomIdentity()
// 	require.NoError(b, err)

// 	h, err := libp2p.New(
// 		libp2p.NoTransports,
// 		libp2p.Transport(inproc.New()),
// 		libp2p.Identity(priv),
// 	)
// 	require.NoError(b, err)
// 	defer h.Close()

// 	ps, err := pubsub.NewGossipSub(ctx, h,
// 		pubsub.WithPeerExchange(true),
// 		pubsub.WithProtocolMatchFn(ProtoMatchFunc()),
// 		pubsub.WithGossipSubProtocols(Subprotocols()))
// 	require.NoError(b, err)

// 	topic, err := ps.Join(topic)
// 	require.NoError(b, err)

// 	group, ctx := errgroup.WithContext(ctx)

// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		group.Go(func() error { return topic.Publish(ctx, []byte(payload)) })
// 	}

// 	require.NoError(b, group.Wait())
// }

// func BenchmarkPubSubCap(b *testing.B) {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	ps, err := pubsub.NewGossipSub(ctx, h,
// 		pubsub.WithPeerExchange(true),
// 		pubsub.WithProtocolMatchFn(ProtoMatchFunc()),
// 		pubsub.WithGossipSubProtocols(Subprotocols()))

// 	require.NoError(b, err)
// 	router := pscap.Server{TopicJoiner: ps}

// 	client := pscap.Router(router.Client())
// 	defer client.Release()

// 	topic, release := client.Join(ctx, payload)
// 	defer release()

// 	group, ctx := errgroup.WithContext(ctx)

// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		group.Go(func() error { return topic.Publish(ctx, []byte(payload)) })
// 	}

// 	require.NoError(b, group.Wait())
// }

// func BenchmarkPubSubCapNetwork(b *testing.B) {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	ps, err := pubsub.NewGossipSub(ctx, h,
// 		pubsub.WithPeerExchange(true),
// 		pubsub.WithProtocolMatchFn(ProtoMatchFunc()),
// 		pubsub.WithGossipSubProtocols(Subprotocols()))
// 	require.NoError(b, err)

// 	router := pscap.Server{TopicJoiner: ps}

// 	left, right := transport.NewPipe(1)
// 	p1, p2 := rpc.NewTransport(left), rpc.NewTransport(right)

// 	conn1 := rpc.NewConn(p1, &rpc.Options{
// 		BootstrapClient: router.Client(),
// 	})
// 	defer conn1.Close()

// 	conn2 := rpc.NewConn(p2, &rpc.Options{})
// 	defer conn2.Close()

// 	client := pscap.Router(conn2.Bootstrap(ctx))
// 	defer client.Release()

// 	topic, release := client.Join(ctx, payload)
// 	defer release()

// 	group, ctx := errgroup.WithContext(ctx)

// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		group.Go(func() error {
// 			return topic.Publish(ctx, []byte(payload))
// 		})
// 	}

// 	require.NoError(b, group.Wait())
// }

// func Proto() protocol.ID {
// 	return protoutil.Join(
// 		ww.Subprotocol(ns),
// 		pubsub.GossipSubID_v11)
// }

// func Features() func(pubsub.GossipSubFeature, protocol.ID) bool {
// 	supportGossip := Matcher()

// 	_, version := protoutil.Split(Proto())
// 	supportsPX := protoutil.Suffix(version)

// 	return func(feat pubsub.GossipSubFeature, proto protocol.ID) bool {
// 		switch feat {
// 		case pubsub.GossipSubFeatureMesh:
// 			return supportGossip.Match(proto)

// 		case pubsub.GossipSubFeaturePX:
// 			return supportsPX.Match(proto)

// 		default:
// 			return false
// 		}
// 	}
// }

// func Subprotocols() ([]protocol.ID, func(pubsub.GossipSubFeature, protocol.ID) bool) {
// 	return []protocol.ID{Proto()}, Features()
// }

// func Matcher() protoutil.MatchFunc {
// 	proto, version := protoutil.Split(pubsub.GossipSubID_v11)
// 	return protoutil.Match(
// 		ww.NewMatcher(ns),
// 		protoutil.Exactly(string(proto)),
// 		protoutil.SemVer(string(version)))
// }

// func ProtoMatchFunc() pubsub.ProtocolMatchFn {
// 	match := Matcher()

// 	return func(local protocol.ID) func(protocol.ID) bool {
// 		if match.Match(local) {
// 			return match.Match
// 		}
// 		panic(fmt.Sprintf("match failed for local protocol %s", local))
// 	}
// }

// func randomIdentity() (crypto.PrivKey, error) {
// 	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
// 	return priv, err
// }
