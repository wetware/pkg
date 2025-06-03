package vat

import (
	"context"
	"strings"

	"capnproto.org/go/capnp/v3/rpc"
	p2p "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/wetware/pkg/util/proto"
)

func DefaultDialOpts(opt ...p2p.Option) []p2p.Option {
	return append([]p2p.Option{
		p2p.NoTransports,
		p2p.NoListenAddrs,
		p2p.Transport(tcp.NewTCPTransport),
		p2p.Transport(quic.NewTransport)},
		opt...)
}

func DialP2P() (local.Host, error) {
	return p2p.New(DefaultDialOpts()...)
}

func DefaultListenOpts(opt ...p2p.Option) []p2p.Option {
	return append([]p2p.Option{
		p2p.NoTransports,
		p2p.Transport(tcp.NewTCPTransport),
		p2p.Transport(quic.NewTransport),
	}, opt...)
}

func ListenP2P(listen ...string) (local.Host, error) {
	return p2p.New(DefaultListenOpts(p2p.ListenAddrStrings(listen...))...)
}

func NewDHT(ctx context.Context, h local.Host, ns string) (*dual.DHT, error) {
	return dual.New(ctx, h,
		dual.LanDHTOption(lanOpt(ns)...),
		dual.WanDHTOption(wanOpt(ns)...))
}

func lanOpt(ns string) []dht.Option {
	return []dht.Option{
		dht.Mode(dht.ModeServer),
		dht.ProtocolPrefix(proto.Root(ns)),
		dht.ProtocolExtension("/lan")}
}

func wanOpt(ns string) []dht.Option {
	return []dht.Option{
		dht.Mode(dht.ModeAuto),
		dht.ProtocolPrefix(proto.Root(ns)),
		dht.ProtocolExtension("/wan")}
}

func Transport(s network.Stream) rpc.Transport {
	if strings.HasSuffix(string(s.Protocol()), "/packed") {
		return rpc.NewPackedStreamTransport(s)
	}

	return rpc.NewStreamTransport(s)
}
