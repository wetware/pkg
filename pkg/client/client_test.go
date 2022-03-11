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
	clapi "github.com/wetware/ww/internal/api/cluster"
	psapi "github.com/wetware/ww/internal/api/pubsub"
	"github.com/wetware/ww/pkg/cap/cluster"
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

	h, err := libp2p.New(
		libp2p.NoListenAddrs,
		libp2p.NoTransports,
		libp2p.ListenAddrStrings("/inproc/~"),
		libp2p.Transport(inproc.New()))
	require.NoError(t, err, "must succeed")
	defer h.Close()

	svr := vat.Network{
		NS:   "test",
		Host: h,
	}
	svr.Export(pubsub.Capability, mockPubSub{})
	svr.Export(cluster.ViewCapability, mockView{})

	clt := newVat()
	defer clt.Host.Close()

	n, err := client.Dialer{
		Vat:  clt,
		Boot: boot.StaticAddrs{*host.InfoFromHost(h)},
	}.Dial(ctx)

	assert.NoError(t, err, "should return without error")
	assert.NotNil(t, n, "should return non-nil node")

	err = n.Bootstrap(ctx)
	assert.NoError(t, err, "should bootstrap successfully")
}

type mockPubSub struct{}

func (mockPubSub) Join(ctx context.Context, call psapi.PubSub_join) error {
	return fmt.Errorf("NOT IMPLEMENTED")
}

func (mockPubSub) Client() *capnp.Client {
	return psapi.PubSub_ServerToClient(mockPubSub{}, nil).Client
}

type mockView struct{}

func (mockView) Client() *capnp.Client {
	return clapi.View_ServerToClient(mockView{}, nil).Client
}

func (mockView) Iter(ctx context.Context, call clapi.View_iter) error {
	return fmt.Errorf("NOT IMPLEMENTED")
}

func (mockView) Lookup(ctx context.Context, call clapi.View_lookup) error {
	return fmt.Errorf("NOT IMPLEMENTED")
}
