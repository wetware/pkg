package client

import (
	"context"
	"math/rand"

	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	protocol "github.com/libp2p/go-libp2p-protocol"
	"github.com/pkg/errors"
)

type callHandler interface {
	Err() error
	Fail(error)
	HandleRPC(context.Context, network.Stream)
}

type session interface {
	Call(context.Context, protocol.ID, callHandler)
}

// terminal is responsible for managing interactive sessions with remote hosts.
//
// TODO:  Connection caching/pooling is in-scope and will be added in the future.
type terminal struct {
	local host.Host
}

// AutoDial returns an arbitrary host session, dialing a new connection if needed.
func (term terminal) AutoDial() session {
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
		return errSession{err: errors.New("unable to dial: no hosts")}
	}

	rand.Shuffle(len(ids), func(i, j int) {
		ids[i], ids[j] = ids[j], ids[i]
	})

	return term.Dial(ids[0])
}

// DialString into a remote host using its string-encoded peer.ID.
// It is equivalent to calling peer.Decode followed by terminal.Dial.
func (term terminal) DialString(id string) session {
	hostID, err := peer.Decode(id)
	if err != nil {
		return errSession{err}
	}

	return term.Dial(hostID)
}

// Dial a host and start a remote session.
func (term terminal) Dial(id peer.ID) session {

	// TODO(connpolicy) calls to dial should tag the connection in the ConnManager.

	return remoteSession{
		local:  term.local,
		remote: id,
	}
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

type remoteSession struct {
	local  host.Host
	remote peer.ID
}

func (sess remoteSession) Call(ctx context.Context, pid protocol.ID, h callHandler) {

	// We're going to be instantiating a lot of different things here.
	// Consider using Fx to populate `h`, or using https://pkg.go.dev/go.uber.org/dig
	// directly.

	s, err := sess.local.NewStream(ctx, sess.remote, pid)
	if err != nil {
		h.Fail(err)
		return
	}

	h.HandleRPC(ctx, s)
}

// errSession represents a session dial that has failed.
// Its methods will report the error in the most appropriate
// fashion for the given call.
//
// errSession implements the standard go `error` interface.
type errSession struct {
	err error
}

func (sess errSession) Call(_ context.Context, _ protocol.ID, h callHandler) {
	h.Fail(sess.err)
}
