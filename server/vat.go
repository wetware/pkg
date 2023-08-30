package server

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/pkg/errors"

	"github.com/wetware/pkg/boot"
	"github.com/wetware/pkg/cluster/pulse"
)

var _ rpc.Network = (*Vat)(nil)

type Vat struct {
	NS   boot.Namespace
	Host local.Host
	Meta pulse.Preparer

	ch chan network.Stream
}

func (vat Vat) String() string {
	return fmt.Sprintf("%s:%s", vat.NS, vat.Host.ID())
}

func (vat Vat) Logger() *slog.Logger {
	return slog.Default().With(
		"ns", vat.NS,
		"peer", vat.Host.ID())
}

// Return the identifier for caller on this network.
func (vat Vat) LocalID() rpc.PeerID {
	return rpc.PeerID{
		Value: vat.Host.ID(),
	}
}

// Connect to another peer by ID. The supplied Options are used
// for the connection, with the values for RemotePeerID and Network
// overridden by the Network.
func (vat Vat) Dial(pid rpc.PeerID, opt *rpc.Options) (*rpc.Conn, error) {
	ctx := context.TODO()

	opt.RemotePeerID = pid
	opt.Network = vat

	peer := pid.Value.(peer.AddrInfo)
	protos := vat.NS.Protocols()

	s, err := vat.Host.NewStream(ctx, peer.ID, protos...)
	if err != nil {
		return nil, err
	}

	conn := rpc.NewConn(transport(s), opt)
	return conn, nil
}

// Accept the next incoming connection on the network, using the
// supplied Options for the connection. Generally, callers will
// want to invoke this in a loop when launching a server.
func (vat Vat) Accept(ctx context.Context, opt *rpc.Options) (*rpc.Conn, error) {
	select {
	case s, ok := <-vat.ch:
		if !ok {
			return nil, errors.New("closed")
		}

		opt.RemotePeerID.Value = peer.AddrInfo{
			ID:    s.Conn().RemotePeer(),
			Addrs: vat.Host.Peerstore().Addrs(s.Conn().RemotePeer()),
		}
		opt.Network = vat

		conn := rpc.NewConn(transport(s), opt)
		return conn, nil

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

func transport(s network.Stream) rpc.Transport {
	if strings.HasSuffix(string(s.Protocol()), "/packed") {
		return rpc.NewPackedStreamTransport(s)
	}

	return rpc.NewStreamTransport(s)
}
