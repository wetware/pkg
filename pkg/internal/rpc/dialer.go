package rpc

import (
	"context"
	"math/rand"

	"github.com/pkg/errors"
	capnp "zombiezen.com/go/capnproto2"
	"zombiezen.com/go/capnproto2/rpc"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	protocol "github.com/libp2p/go-libp2p-core/protocol"
)

// Dialer .
type Dialer interface {
	Dial(context.Context, host.Host, []protocol.ID) Client
}

// DialPeer opens a transport connection to the specified peer.
type DialPeer peer.ID

// Dial decodes the peer ID string and then opens a transport to the specified host.
func (d DialPeer) Dial(ctx context.Context, h host.Host, pid []protocol.ID) Client {
	return Dial(ctx, h, peer.ID(d), pid)
}

// DialString interprets a string as a peer.ID
type DialString string

// Dial decodes the peer ID string and then opens a transport to the specified host.
func (d DialString) Dial(ctx context.Context, h host.Host, pid []protocol.ID) Client {
	id, err := peer.Decode(string(d))
	if err != nil {
		return errclient(err, "decode id")
	}

	return Dial(ctx, h, id, pid)
}

// Dial opens a transport to the specified peer
func Dial(ctx context.Context, h host.Host, id peer.ID, pid []protocol.ID) Client {
	s, err := h.NewStream(ctx, id, pid...)
	if err != nil {
		return errclient(err, "open stream")
	}

	// TODO(performance):  packed stream transport
	return Client{
		Peer:   id,
		Client: rpc.NewConn(rpc.StreamTransport(s)).Bootstrap(ctx),
	}
}

// AutoDial opens a transport connection to an arbitrary peer.
type AutoDial struct{}

// Dial an arbitrary peer
func (d AutoDial) Dial(ctx context.Context, h host.Host, pid []protocol.ID) Client {
	id, err := d.selectPeer(ctx, h)
	if err != nil {
		return errclient(err, "select peer")
	}

	return Dial(ctx, h, id, pid)
}

func (d AutoDial) selectPeer(ctx context.Context, h host.Host) (id peer.ID, err error) {
	for _, source := range []func(host.Host) (peer.ID, error){
		d.fromConns,
		d.fromPeerstore,
	} {
		if id, err = source(h); err != nil || id != "" {
			break
		}
	}

	if err == nil && id == "" {
		err = errors.New("local host is orphaned")
	}

	return
}

func (d AutoDial) fromConns(h host.Host) (peer.ID, error) {
	cs := h.Network().Conns()
	hs := make(peer.IDSlice, len(cs))
	for i, conn := range cs {
		hs[i] = conn.RemotePeer()
	}

	return pick(hs), nil
}

func (d AutoDial) fromPeerstore(h host.Host) (peer.ID, error) {
	all := h.Peerstore().Peers()
	hosts := all[0:]
	for _, id := range all {
		if id == h.ID() {
			continue
		}

		hosts = append(hosts, id)
	}

	return pick(hosts), nil
}

func pick(ids peer.IDSlice) peer.ID {
	switch len(ids) {
	case 0:
		return ""
	case 1:
	default:
		rand.Shuffle(len(ids), func(i, j int) {
			ids[i], ids[j] = ids[j], ids[i]
		})
	}

	return ids[0]
}

func errclient(err error, msg string) Client {
	return Client{
		Client: capnp.ErrorClient(errors.Wrap(err, msg)),
	}
}
