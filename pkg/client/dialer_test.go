package client_test

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	swarm "github.com/libp2p/go-libp2p-swarm"
	inproc "github.com/lthibault/go-libp2p-inproc-transport"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wetware/casm/pkg/boot"
	"github.com/wetware/ww/pkg/client"
	"github.com/wetware/ww/pkg/vat"
)

func TestDialer(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("NoPeers", func(t *testing.T) {
		t.Parallel()

		vat := newVat()
		defer vat.Host.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // NOTE:  eagerly canceled

		n, err := client.Dialer{
			Vat:  vat,
			Boot: boot.StaticAddrs(nil),
		}.Dial(ctx)

		assert.EqualError(t, err, "bootstrap failed: no peers found")
		assert.Nil(t, n, "should return nil client node")
	})

	t.Run("PeerConnectionFailure", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		id, err := peer.Decode("QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N")
		require.NoError(t, err, "test invariant violated:  peer.ID must be valid")

		info := peer.AddrInfo{
			ID: id,
			Addrs: []ma.Multiaddr{
				ma.StringCast("/inproc/does-not-exist"),
			},
		}

		vat := newVat()
		defer vat.Host.Close()

		n, err := client.Dialer{
			Vat:  vat,
			Boot: boot.StaticAddrs{info},
		}.Dial(ctx)

		derr := new(swarm.DialError)
		assert.ErrorAs(t, err, &derr,
			"should return swarm.DialError")
		assert.Len(t, derr.DialErrors, 1,
			"DialError should wrap exactly one swarm.TransportError")

		assert.Equal(t,
			info.Addrs[0],
			derr.DialErrors[0].Address,
			"address should match peer.AddrInfo")

		assert.Nil(t, n, "should return nil client node")
	})

	t.Run("StreamNegotiationFailure", func(t *testing.T) {
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

		clt := newVat()
		defer clt.Host.Close()

		n, err := client.Dialer{
			Vat:  clt,
			Boot: boot.StaticAddrs{*host.InfoFromHost(h)},
		}.Dial(ctx)

		assert.EqualError(t, err, "protocol not supported")
		assert.Nil(t, n, "should return nil client node")
	})
}

func newVat() vat.Network {
	h, err := libp2p.New(
		libp2p.NoListenAddrs,
		libp2p.NoTransports,
		libp2p.Transport(inproc.New()))
	if err != nil {
		panic(err)
	}

	return vat.Network{
		NS:   "test",
		Host: h,
	}
}
