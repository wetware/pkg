package client

import (
	"context"
	"math/rand"

	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	protocol "github.com/libp2p/go-libp2p-protocol"
	"github.com/pkg/errors"
	capnp "zombiezen.com/go/capnproto2"
	"zombiezen.com/go/capnproto2/rpc"
)

// terminal is responsible for managing interactive sessions with remote hosts.
//
// TODO:  Connection caching/pooling is in-scope and will be added in the future.
type terminal struct {
	local host.Host
}

// Dial a host and start a remote session.
func (term terminal) Dial(ctx context.Context, pid protocol.ID, id peer.ID) capnp.Client {
	s, err := term.local.NewStream(ctx, id, pid)
	if err != nil {
		return capnp.ErrorClient(err)
	}

	return rpc.NewConn(rpc.StreamTransport(s)).Bootstrap(ctx)
}

// AutoDial returns an arbitrary host session, dialing a new connection if needed.
func (term terminal) AutoDial(ctx context.Context, pid protocol.ID) capnp.Client {
	var ids peer.IDSlice
	for _, source := range []func() peer.IDSlice{
		term.fromConns,
		term.fromPeerstore,
	} {
		if ids = source(); ids != nil {
			break
		}
	}

	if len(ids) == 0 {
		return capnp.ErrorClient(errors.New("unable to dial: no hosts"))
	}

	rand.Shuffle(len(ids), func(i, j int) {
		ids[i], ids[j] = ids[j], ids[i]
	})

	return term.Dial(ctx, pid, ids[0])
}

func (term terminal) fromConns() peer.IDSlice {
	cs := term.local.Network().Conns()
	hs := make(peer.IDSlice, len(cs))
	for i, conn := range cs {
		hs[i] = conn.RemotePeer()
	}

	return hs
}

func (term terminal) fromPeerstore() peer.IDSlice {
	all := term.local.Peerstore().Peers()
	hosts := all[0:]
	for _, id := range all {
		if id == term.local.ID() {
			continue
		}

		hosts = append(hosts, id)
	}

	return hosts
}
