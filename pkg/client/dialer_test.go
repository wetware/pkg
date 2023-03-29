package client_test

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/discovery"
	p2p_host "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/net/swarm"
	inproc "github.com/lthibault/go-libp2p-inproc-transport"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multistream"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/boot"
	"github.com/wetware/ww/pkg/client"
	"github.com/wetware/ww/pkg/host"
)

func TestDialer(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("BootError", func(t *testing.T) {
		t.Parallel()

		vat := newVat()
		defer vat.Host.Close()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		n, err := client.Dialer{
			Vat:  vat,
			Boot: bootErr{},
		}.Dial(ctx)

		assert.ErrorAs(t, err, new(bootErr),
			"should wrap errors from bootstrapper")
		assert.Nil(t, n, "should return nil client node")
	})

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

		assert.ErrorIs(t, err, client.ErrNoPeers)
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
			Boot: boot.StaticAddrs{*p2p_host.InfoFromHost(h)},
		}.Dial(ctx)

		var want multistream.ErrNotSupported[protocol.ID]
		assert.ErrorAs(t, err, &want)
		assert.Contains(t, want.Protos, protocol.ID("/casm/0.0.0/test/host/packed"),
			"packed protocol missing")
		assert.Contains(t, want.Protos, protocol.ID("/casm/0.0.0/test/host"),
			"standard protocol missing")
		assert.Nil(t, n, "should return nil client node")
	})

	t.Run("NamespaceMismatch", func(t *testing.T) {
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

		clt.NS = "wrong.namespace"

		svr := casm.Vat{
			NS:   "test",
			Host: h,
		}
		svr.Export(host.Capability, host.Server{})

		n, err := client.Dialer{
			Vat:  clt,
			Boot: boot.StaticAddrs{*p2p_host.InfoFromHost(h)},
		}.Dial(ctx)

		assert.ErrorIs(t, err, casm.ErrInvalidNS)
		assert.Nil(t, n, "should return nil client node")
	})

	t.Run("Success", func(t *testing.T) {
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

		svr := casm.Vat{
			NS:   "test",
			Host: h,
		}
		svr.Export(host.Capability, host.Server{})

		clt := newVat()
		defer clt.Host.Close()

		n, err := client.Dialer{
			Vat:  clt,
			Boot: boot.StaticAddrs{*p2p_host.InfoFromHost(h)},
		}.Dial(ctx)

		require.NoError(t, err, "should return without error")
		require.NotNil(t, n, "should return non-nil node")
	})
}

func newVat() casm.Vat {
	h, err := libp2p.New(
		libp2p.NoListenAddrs,
		libp2p.NoTransports,
		libp2p.Transport(inproc.New()))
	if err != nil {
		panic(err)
	}

	return casm.Vat{
		NS:   "test",
		Host: h,
	}
}

type bootErr struct{}

func (bootErr) Error() string { return "test" }

func (bootErr) FindPeers(context.Context, string, ...discovery.Option) (<-chan peer.AddrInfo, error) {
	return nil, bootErr{}
}
