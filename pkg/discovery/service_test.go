package discovery_test

import (
	"context"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	inproc "github.com/lthibault/go-libp2p-inproc-transport"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	api "github.com/wetware/ww/internal/api/service"
	"github.com/wetware/ww/pkg/discovery"
	pscap "github.com/wetware/ww/pkg/pubsub"
)

func TestDiscover(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gs, release := newGossipSub(ctx)
	defer release()

	ps := (&pscap.Router{TopicJoiner: gs}).PubSub()
	defer ps.Release()

	// create server
	server := discovery.DiscoveryServiceServer{
		Joiner: ps,
	}
	// create 1 client
	client := &discovery.DiscoveryService{api.DiscoveryService_ServerToClient(&server)}
	defer client.Release()
	// advertise service in 1 client

	const (
		serviceName = "service.test"
		infoN       = 2
	)

	// advertise and find
	provider, release := client.Provider(ctx, serviceName)
	defer release()

	infos := generateInfos(infoN)
	_, release = provider.Provide(ctx, infos)
	defer release()

	time.Sleep(time.Second) // give time for the provider to set

	finder, release := client.Locator(ctx, serviceName)
	defer release()

	providers, release := finder.FindProviders(ctx)
	defer release()

	gotInfos := make([]peer.AddrInfo, 0, infoN)

	for i := 0; i < infoN; i++ {
		info, ok := providers.Next()
		require.True(t, ok)
		gotInfos = append(gotInfos, info)
	}

	require.EqualValues(t, infos, gotInfos)
}

func generateInfos(n int) []peer.AddrInfo {
	infos := make([]peer.AddrInfo, 0, n)
	addr, _ := ma.NewMultiaddr("/ip4/0.0.0.0")
	for i := 0; i < n; i++ {
		info := peer.AddrInfo{
			ID:    peer.ID(strconv.Itoa(rand.Int())),
			Addrs: []ma.Multiaddr{addr},
		}
		infos = append(infos, info)
	}

	return infos
}

func generateRequest() api.Message_Request {
	_, seg := capnp.NewSingleSegmentMessage(nil)
	request, _ := api.NewMessage_Request(seg)
	return request
}

func generateResponse(infos []peer.AddrInfo) api.Message_Response {
	_, seg := capnp.NewSingleSegmentMessage(nil)
	response, _ := api.NewMessage_Response(seg)
	capInfos, _ := discovery.ToCapInfoList(infos)
	response.SetAddrs(capInfos)
	return response
}

func newGossipSub(ctx context.Context) (*pubsub.PubSub, func()) {
	h := newTestHost()

	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		panic(err)
	}

	return ps, func() { h.Close() }
}

func newTestHost() host.Host {
	h, err := libp2p.New(
		libp2p.NoListenAddrs,
		libp2p.NoTransports,
		libp2p.Transport(inproc.New()),
		libp2p.ListenAddrStrings("/inproc/~"))
	if err != nil {
		panic(err)
	}

	return h
}
