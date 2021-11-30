package client

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/pnet"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	ww "github.com/wetware/ww/pkg"
	"go.uber.org/fx"
)

const userAgent = "ww.node.client:%s"

var defaultHostOpt = []libp2p.Option{
	libp2p.NoListenAddrs,
	libp2p.NoTransports,
	libp2p.Transport(libp2pquic.NewTransport),
	libp2p.UserAgent(fmt.Sprintf(userAgent, ww.Version)),
}

type HostFactory interface {
	New(context.Context) (host.Host, error)
}

type BasicHostFactory struct {
	fx.In `ignore-unexported:"true"`

	Secret pnet.PSK                `optional:"true"`
	Auth   connmgr.ConnectionGater `optional:"true"`

	routing *routingHook
}

func (f *BasicHostFactory) New(ctx context.Context) (host.Host, error) {
	var opt = make([]libp2p.Option, len(defaultHostOpt))
	copy(opt, defaultHostOpt)

	if f.Secret != nil {
		opt = append(opt, libp2p.PrivateNetwork(f.Secret))
	}

	if f.Auth != nil {
		opt = append(opt, libp2p.ConnectionGater(f.Auth))
	}

	if f.routing != nil {
		opt = append(opt, f.routing.Option())
		f.routing = nil // make 'f' reusable
	}

	return libp2p.New(ctx, opt...)
}
