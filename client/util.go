package client

import (
	p2p "github.com/libp2p/go-libp2p"
	local "github.com/libp2p/go-libp2p/core/host"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
)

func NetConfig(opt ...p2p.Option) []p2p.Option {
	return append([]p2p.Option{
		p2p.NoTransports,
		p2p.NoListenAddrs,
		p2p.Transport(tcp.NewTCPTransport),
		p2p.Transport(quic.NewTransport)},
		opt...)
}

func DialP2P() (local.Host, error) {
	return p2p.New(NetConfig()...)
}