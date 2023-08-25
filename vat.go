package ww

import (
	"context"
	"fmt"
	"net"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"

	"github.com/wetware/pkg/server"
	"github.com/wetware/pkg/system"
	"golang.org/x/exp/slog"
	"golang.org/x/sync/errgroup"
)

type Addr struct {
	NS  string
	Vat peer.ID
}

func (addr Addr) Network() string {
	return addr.NS
}

func (addr Addr) String() string {
	return addr.Vat.String()
}

type ClientProvider[T ~capnp.ClientKind] interface {
	Client() T
}

type RPCDialer interface {
	DialRPC(context.Context, net.Addr, *rpc.Options) (*rpc.Conn, error)
}

type Vat[T ~capnp.ClientKind] struct {
	Ctx    context.Context
	Addr   *Addr
	Host   host.Host
	Dialer RPCDialer
	Export ClientProvider[T]

	in <-chan *rpc.Conn
}

func (vat Vat[T]) String() string {
	return fmt.Sprintf("%s:%s", vat.Addr.NS, vat.Addr.Vat)
}

// Return the identifier for caller on this network.
func (vat Vat[T]) LocalID() rpc.PeerID {
	return rpc.PeerID{
		Value: vat.Addr,
	}
}

func (vat Vat[T]) Serve(ctx context.Context) error {
	config := server.Config{
		Logger: slog.Default(),
		NS:     vat.Addr.NS,
	}

	server, err := config.NewServer(ctx, vat.Host)
	if err != nil {
		return fmt.Errorf("new server: %w", err)
	}

	in := make(chan *rpc.Conn)
	vat.in = in

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		defer close(in)

		client := capnp.Client(vat.Export.Client())
		defer client.Release()

		reporter := &system.ErrorReporter{
			Logger: config.Logger,
		}

		opts := &rpc.Options{
			BootstrapClient: client.AddRef(),
			ErrorReporter:   reporter,
		}

		for {
			conn, err := vat.Accept(ctx, opts)
			if err != nil {
				return fmt.Errorf("accept: %w", err)
			}

			select {
			case in <- conn:
			case <-ctx.Done():
				go conn.Close()
			}
		}
	})
	g.Go(func() error {
		err := server.Serve(ctx, vat.Host, config)
		return fmt.Errorf("serve: %w", err)
	})

	return g.Wait()
}

// Connect to another peer by ID. The supplied Options are used
// for the connection, with the values for RemotePeerID and Network
// overridden by the Network.
func (vat Vat[T]) Dial(pid rpc.PeerID, opt *rpc.Options) (*rpc.Conn, error) {
	opt.RemotePeerID = pid
	opt.Network = vat

	addr := pid.Value.(net.Addr)
	return vat.Dialer.DialRPC(vat.Ctx, addr, opt)
}

// Accept the next incoming connection on the network, using the
// supplied Options for the connection. Generally, callers will
// want to invoke this in a loop when launching a server.
func (vat Vat[T]) Accept(ctx context.Context, opt *rpc.Options) (*rpc.Conn, error) {
	select {
	case conn, ok := <-vat.in:
		if ok {
			return conn, nil
		}

		return nil, rpc.ErrConnClosed

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Introduce the two connections, in preparation for a third party
// handoff. Afterwards, a Provide messsage should be sent to
// provider, and a ThirdPartyCapId should be sent to recipient.
func (vat Vat[T]) Introduce(provider, recipient *rpc.Conn) (rpc.IntroductionInfo, error) {
	return rpc.IntroductionInfo{}, errors.New("NOT IMPLEMENTED")
}

// Given a ThirdPartyCapID, received from introducedBy, connect
// to the third party. The caller should then send an Accept
// message over the returned Connection.
func (vat Vat[T]) DialIntroduced(capID rpc.ThirdPartyCapID, introducedBy *rpc.Conn) (*rpc.Conn, rpc.ProvisionID, error) {
	return nil, rpc.ProvisionID{}, errors.New("NOT IMPLEMENTED")
}

// Given a RecipientID received in a Provide message via
// introducedBy, wait for the recipient to connect, and
// return the connection formed. If there is already an
// established connection to the relevant Peer, this
// SHOULD return the existing connection immediately.
func (vat Vat[T]) AcceptIntroduced(recipientID rpc.RecipientID, introducedBy *rpc.Conn) (*rpc.Conn, error) {
	return nil, errors.New("NOT IMPLEMENTED")
}
