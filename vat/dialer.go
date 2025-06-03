package vat

import (
	"context"
	"fmt"

	"capnproto.org/go/capnp/v3/rpc"
	"golang.org/x/exp/slog"

	"github.com/libp2p/go-libp2p/core/discovery"
	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	api "github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/boot"
	"github.com/wetware/pkg/system"
	"github.com/wetware/pkg/util/proto"
)

type Dialer struct {
	Host    local.Host
	Account auth.Signer
}

func (d Dialer) DialDiscover(ctx context.Context, disc discovery.Discoverer, ns string) (auth.Session, error) {
	peers, err := disc.FindPeers(ctx, ns)
	if err != nil {
		return auth.Session{}, fmt.Errorf("find peers: %w", err)
	}

	err = boot.ErrNoPeers
	for info := range peers {
		if err = d.Host.Connect(ctx, info); err != nil {
			continue
		}

		return d.Dial(ctx, info, proto.Namespace(ns)...)
	}

	return auth.Session{}, err
}

func (d Dialer) Dial(ctx context.Context, addr peer.AddrInfo, protos ...protocol.ID) (auth.Session, error) {
	conn, err := d.DialRPC(ctx, addr, protos...)
	if err != nil {
		return auth.Session{}, fmt.Errorf("dial: %w", err)
	}

	client := conn.Bootstrap(ctx)
	if err := client.Resolve(ctx); err != nil {
		return auth.Session{}, fmt.Errorf("bootstrap: %w", err)
	}

	term := api.Terminal(client)
	defer term.Release()

	f, release := term.Login(ctx, func(call api.Terminal_login_Params) error {
		return call.SetAccount(d.Account.Account())
	})
	defer release()

	res, err := f.Struct()
	if err != nil {
		return auth.Session{}, err
	}

	sess, err := res.Session()
	if err != nil {
		return auth.Session{}, err
	}

	return auth.Session(sess).AddRef(), nil
}

func (d Dialer) DialRPC(ctx context.Context, addr peer.AddrInfo, protos ...protocol.ID) (*rpc.Conn, error) {
	s, err := d.DialP2P(ctx, addr, protos...)
	if err != nil {
		return nil, err
	}

	conn := rpc.NewConn(Transport(s), &rpc.Options{
		BootstrapClient: d.Account.Client(),
		ErrorReporter: system.ErrorReporter{
			Logger: slog.Default().With(
				"stream", s.ID(),
				"remote", addr.ID,
				"proto", s.Protocol()),
		},
	})

	return conn, nil
}

func (d Dialer) DialP2P(ctx context.Context, addr peer.AddrInfo, protos ...protocol.ID) (network.Stream, error) {
	if len(addr.Addrs) > 0 {
		if err := d.Host.Connect(ctx, addr); err != nil {
			return nil, fmt.Errorf("dial %s: %w", addr.ID, err)
		}
	}

	return d.Host.NewStream(ctx, addr.ID, protos...)
}

func (d Dialer) DialConn(ctx context.Context, conn *rpc.Conn) (auth.Session, error) {
	client := conn.Bootstrap(ctx)
	if err := client.Resolve(ctx); err != nil {
		return auth.Session{}, fmt.Errorf("bootstrap: %w", err)
	}

	term := api.Terminal(client)
	defer term.Release()

	f, release := term.Login(ctx, func(call api.Terminal_login_Params) error {
		return call.SetAccount(d.Account.Account())
	})
	defer release()

	res, err := f.Struct()
	if err != nil {
		return auth.Session{}, err
	}

	sess, err := res.Session()
	if err != nil {
		return auth.Session{}, err
	}

	return auth.Session(sess).AddRef(), nil
}
