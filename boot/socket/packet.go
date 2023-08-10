package socket

import (
	"net"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/wetware/pkg/api/boot"
)

type Request struct {
	Record
	NS   string
	From net.Addr
}

func (r Request) IsSurvey() bool {
	return r.asPacket().Which() == boot.Packet_Which_survey
}

func (r Request) Distance() (dist uint8) {
	if r.IsSurvey() {
		dist = r.asPacket().Survey().Distance()
	}

	return
}

type Response struct {
	Record
	NS   string
	From net.Addr
}

func (r Response) Addrs() ([]ma.Multiaddr, error) {
	addrs, err := r.asPacket().Response().Addrs()
	if err != nil {
		return nil, err
	}

	var (
		b  []byte
		as = make([]ma.Multiaddr, addrs.Len())
	)

	for i := range as {
		if b, err = addrs.At(i); err != nil {
			break
		}

		if as[i], err = ma.NewMultiaddrBytes(b); err != nil {
			break
		}
	}

	return as, err
}

func (r Response) Bind(info *peer.AddrInfo) (err error) {
	if info.ID, err = r.Peer(); err == nil {
		info.Addrs, err = r.Addrs()
	}

	return
}
