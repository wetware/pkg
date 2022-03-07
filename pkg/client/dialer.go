package client

import (
	"context"
	"errors"
	"fmt"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/wetware/casm/pkg/boot"
	"github.com/wetware/ww/pkg/cap/cluster"
	"github.com/wetware/ww/pkg/cap/pubsub"
	"github.com/wetware/ww/pkg/vat"
)

type Addr string

func (addr Addr) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	info, err := peer.AddrInfoFromString(string(addr))
	if err != nil {
		return nil, err
	}

	return boot.StaticAddrs{*info}.FindPeers(ctx, ns, opt...)
}

type Dialer struct {
	Vat  vat.Network
	Boot discovery.Discoverer
}

// func NewDialer(vat vat.Network, opt ...Option) Dialer {
// 	var d = Dialer{vat: vat}
// 	for _, option := range withDefault(opt) {
// 		option(&d)
// 	}
// 	return d
// }

// Dial is a convenience function that joins a cluster using the
// supplied address string.
//
// See Dialer.Dial for an important notice about the lifetime of
// ctx.
func Dial(ctx context.Context, vat vat.Network, a Addr) (*Node, error) {
	return Dialer{Vat: vat, Boot: a}.Dial(ctx)
}

// Dial creates a client and connects it to a cluster.
//
// Note that 'ctx' MUST NOT be canceled unless Dial's error is
// non-nil, or a previous call to Bootstrap has returned. Failure
// to abide by this rule may cause Node's underlying capabilities
// to fail.
func (d Dialer) Dial(ctx context.Context) (*Node, error) {
	n := &Node{vat: d.Vat, conns: make([]*rpc.Conn, 0)}

	conn, err := d.join(ctx, pubsub.Capability)
	if err != nil {
		return nil, err
	}
	n.ps = pubsub.PubSub{Client: conn.Bootstrap(ctx)}
	n.conns = append(n.conns, conn)

	conn, err = d.join(ctx, cluster.ViewCapability)
	if err != nil {
		n.ps.Release()
		return nil, err
	}
	n.view = cluster.View{Client: conn.Bootstrap(ctx)}
	n.conns = append(n.conns, conn)

	return n, nil
}

func (d Dialer) join(ctx context.Context, cap vat.Capability) (conn *rpc.Conn, err error) {
	var peers <-chan peer.AddrInfo
	if peers, err = d.Boot.FindPeers(ctx, d.Vat.NS); err != nil {
		return nil, fmt.Errorf("discover: %w", err)
	}
	for info := range peers {
		conn, err = d.Vat.Connect(ctx, info, cap)
		if err == nil {
			break
		}
	}

	// no peers discovered?
	if conn == nil && err == nil {
		err = errors.New("bootstrap failed: no peers found")
	}

	return
}
