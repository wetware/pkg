package client

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/wetware/pkg/util/proto"
)

var ErrNoPeers = errors.New("no peers")

type Addr struct {
	net.Addr
	Protos []protocol.ID
}

// PeerDialer can resolve
type PeerDialer interface {
	DialPeer(context.Context, *Addr) (network.Stream, error)
}

type Config struct {
	PeerDialer PeerDialer
}

func (d Config) DialRPC(ctx context.Context, addr net.Addr, opts *rpc.Options) (*rpc.Conn, error) {
	peer := &Addr{
		Addr:   addr,
		Protos: proto.Namespace(addr.Network()),
		// Get a set of Wetware subprotocols that we can try to dial.   These
		// will negotiate things like Cap'n Proto schema version, Cap'n Proto
		// bit-packing and LZ4 compression.
	}

	s, err := d.PeerDialer.DialPeer(ctx, peer)
	if err != nil {
		return nil, fmt.Errorf("dial %v: %w", addr, err)
	}

	conn := rpc.NewConn(transport(s), opts)
	return conn, nil
}

// boot, err := d.newBootstrapper(h)
// if err != nil {
// 	return "", fmt.Errorf("bootstrap: %w", err)
// }
// defer boot.Close()

// var peers <-chan peer.AddrInfo
// if peers, err = boot.FindPeers(ctx, d.NS); err != nil {
// 	return "", fmt.Errorf("discover: %w", err)
// }

// for info := range peers {
// 	if err = h.Connect(ctx, info); err == nil {
// 		return info.ID, nil
// 	}
// }

// // no peers?
// if err == nil {
// 	err = ErrNoPeers
// }

// return "", err

func transport(s network.Stream) rpc.Transport {
	if strings.HasSuffix(string(s.Protocol()), "/packed") {
		return rpc.NewPackedStreamTransport(s)
	}

	return rpc.NewStreamTransport(s)
}
