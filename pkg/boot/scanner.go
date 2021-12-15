package boot

import (
	"context"
	"fmt"
	"net"

	"github.com/libp2p/go-libp2p-core/record"
)

type Dialer interface {
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

type ScanStrategy interface {
	Scan(context.Context, Dialer, record.Record) (*record.Envelope, error)
}

type Handler interface {
	Scan(net.Conn, record.Record) (*record.Envelope, error)
}

type ScanSubnet struct {
	Port int
	CIDR
	Handler Handler
}

func (iter *ScanSubnet) Network() string {
	if iter.CIDR.subnet == nil {
		return ""
	}

	return iter.CIDR.subnet.Network()
}

func (iter *ScanSubnet) String() string {
	var ip = make(net.IP, 4)
	iter.CIDR.Scan(ip)
	return fmt.Sprintf("%s:%d",
		ip.String(),
		iter.Port)
}

func (iter *ScanSubnet) Scan(ctx context.Context, d Dialer, r record.Record) (msg *record.Envelope, err error) {
	var conn net.Conn
	for iter.CIDR.Reset(); iter.More(); iter.Next() {
		if conn, err = iter.dial(ctx, d); err != nil {
			break
		}

		if msg, err = iter.Handler.Scan(conn, r); err != nil {
			break
		}
	}

	return
}

func (iter *ScanSubnet) dial(ctx context.Context, d Dialer) (net.Conn, error) {
	var ip [4]byte
	iter.CIDR.Scan(ip[:])

	return d.DialContext(ctx,
		iter.Network(),
		fmt.Sprintf("%s:%d", ip, iter.Port))
}
