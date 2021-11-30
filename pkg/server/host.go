package server

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/pnet"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	ww "github.com/wetware/ww/pkg"
	"go.uber.org/fx"
)

const userAgent = "ww.node.server:%s"

var defaultHostOpt = []libp2p.Option{
	libp2p.NoTransports,
	libp2p.Transport(libp2pquic.NewTransport),
	libp2p.UserAgent(fmt.Sprintf(userAgent, ww.Version)),
}

type HostFactory interface {
	New(context.Context) (host.Host, error)

	SetListenAddrs(...string)
	SetSecret(pnet.PSK)
	SetAuth(connmgr.ConnectionGater)
}

// RoutedHostFactory creates a host with DHT routing enabled.
//
// Instances of 'RoutedHostFactory' MUST NOT be shared across
// nodes.
type RoutedHostFactory struct {
	fx.In `ignore-unexported:"true"`

	ListenAddrs []string                `optional:"true"`
	Secret      pnet.PSK                `optional:"true"`
	Auth        connmgr.ConnectionGater `optional:"true"`
	PrivKey     crypto.PrivKey          `optional:"true"`

	routing *routingHook
}

func (f *RoutedHostFactory) New(ctx context.Context) (host.Host, error) {
	var opt = make([]libp2p.Option, len(defaultHostOpt))
	copy(opt, defaultHostOpt)

	if f.PrivKey != nil {
		opt = append(opt, libp2p.Identity(f.PrivKey))
	}

	if len(f.ListenAddrs) > 0 {
		opt = append(opt, libp2p.ListenAddrStrings(f.ListenAddrs...))
	} else {
		opt = append(opt, libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/udp/2020/quic",
			"/ip6/::/udp/2020/quic"))
	}

	if f.Secret != nil {
		opt = append(opt, libp2p.PrivateNetwork(f.Secret))
	}

	if f.Auth != nil {
		opt = append(opt, libp2p.ConnectionGater(f.Auth))
	}

	if f.routing != nil {
		opt = append(opt, f.routing.Option())
	}

	return libp2p.New(ctx, opt...)
}

func (f *RoutedHostFactory) SetListenAddrs(ss ...string)       { f.ListenAddrs = ss }
func (f *RoutedHostFactory) SetSecret(s pnet.PSK)              { f.Secret = s }
func (f *RoutedHostFactory) SetAuth(a connmgr.ConnectionGater) { f.Auth = a }
