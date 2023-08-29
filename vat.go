package ww

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/pkg/errors"
	"github.com/wetware/pkg/server"
)

type Addr struct {
	NS    string
	Peer  peer.ID
	Proto []protocol.ID
}

func (addr Addr) Network() string {
	return addr.NS
}

func (addr Addr) String() string {
	return addr.Peer.String()
}

type ClientProvider[T ~capnp.ClientKind] interface {
	Client() (T, io.Closer)
}

type Dialer interface {
	DialRPC(context.Context, net.Addr, ...protocol.ID) (*rpc.Conn, error)
}

type Vat struct {
	Addr   *Addr
	Host   host.Host
	Dialer Dialer
	Server server.Config
	// Export ClientProvider[T]

	in chan *rpc.Conn
}

func (vat Vat) String() string {
	return fmt.Sprintf("%s:%s", vat.Addr.NS, vat.Addr.Peer)
}

func (vat Vat) Logger() *slog.Logger {
	return slog.Default().With(
		"ns", vat.Addr.NS,
		"peer", vat.Addr.Peer,
		"proto", vat.Addr.Proto)
}

// Return the identifier for caller on this network.
func (vat Vat) LocalID() rpc.PeerID {
	return rpc.PeerID{
		Value: vat.Addr,
	}
}

// Connect to another peer by ID. The supplied Options are used
// for the connection, with the values for RemotePeerID and Network
// overridden by the Network.
func (vat Vat) Dial(pid rpc.PeerID, opt *rpc.Options) (*rpc.Conn, error) {
	opt.RemotePeerID = pid
	opt.Network = vat

	addr := pid.Value.(*Addr)
	return vat.Dialer.DialRPC(context.TODO(), addr, vat.Addr.Proto...)
}

// Accept the next incoming connection on the network, using the
// supplied Options for the connection. Generally, callers will
// want to invoke this in a loop when launching a server.
func (vat Vat) Accept(ctx context.Context, opt *rpc.Options) (*rpc.Conn, error) {
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
func (vat Vat) Introduce(provider, recipient *rpc.Conn) (rpc.IntroductionInfo, error) {
	return rpc.IntroductionInfo{}, errors.New("NOT IMPLEMENTED")
}

// Given a ThirdPartyCapID, received from introducedBy, connect
// to the third party. The caller should then send an Accept
// message over the returned Connection.
func (vat Vat) DialIntroduced(capID rpc.ThirdPartyCapID, introducedBy *rpc.Conn) (*rpc.Conn, rpc.ProvisionID, error) {
	return nil, rpc.ProvisionID{}, errors.New("NOT IMPLEMENTED")
}

// Given a RecipientID received in a Provide message via
// introducedBy, wait for the recipient to connect, and
// return the connection formed. If there is already an
// established connection to the relevant Peer, this
// SHOULD return the existing connection immediately.
func (vat Vat) AcceptIntroduced(recipientID rpc.RecipientID, introducedBy *rpc.Conn) (*rpc.Conn, error) {
	return nil, errors.New("NOT IMPLEMENTED")
}
