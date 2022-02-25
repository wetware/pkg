package client_test

import (
	"context"
	"fmt"
	"testing"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	inproc "github.com/lthibault/go-libp2p-inproc-transport"

	"github.com/wetware/casm/pkg/boot"
	api "github.com/wetware/ww/internal/api/pubsub"
	"github.com/wetware/ww/pkg/cap/pubsub"
	"github.com/wetware/ww/pkg/client"
	"github.com/wetware/ww/pkg/vat"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientServerIntegration(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h, err := libp2p.New(ctx,
		libp2p.NoListenAddrs,
		libp2p.NoTransports,
		libp2p.ListenAddrStrings("/inproc/~"),
		libp2p.Transport(inproc.New()))
	require.NoError(t, err, "must succeed")

	vat.Network{
		NS:   "test",
		Host: h,
	}.Export(pubsub.Capability, mockPubSub{})

	n, err := client.Dialer{
		Vat:  newVat(ctx),
		Boot: boot.StaticAddrs{*host.InfoFromHost(h)},
	}.Dial(ctx)

	assert.NoError(t, err, "should return without error")
	assert.NotNil(t, n, "should return non-nil node")

	err = n.Bootstrap(ctx)
	assert.NoError(t, err, "should bootstrap successfully")
}

type mockPubSub struct{}

func (mockPubSub) Join(ctx context.Context, call api.PubSub_join) error {
	return fmt.Errorf("NOT IMPLEMENTED")
}

func (mockPubSub) Client() *capnp.Client {
	return api.PubSub_ServerToClient(mockPubSub{}, nil).Client
}
