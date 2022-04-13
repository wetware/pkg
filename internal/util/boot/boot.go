package bootutil

import (
	"errors"
	"io"
	"net"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/urfave/cli/v2"
	"github.com/wetware/casm/pkg/boot/crawl"
	"github.com/wetware/casm/pkg/boot/socket"
	"github.com/wetware/casm/pkg/boot/survey"
	logutil "github.com/wetware/ww/internal/util/log"
)

// ErrUnknownBootProto is returned when the multiaddr passed
// to Parse does not contain a recognized boot protocol.
var ErrUnknownBootProto = errors.New("unknown boot protocol")

type DiscoveryService interface {
	discovery.Discovery
	io.Closer
}

func Dial(c *cli.Context, h host.Host) (DiscoveryService, error) {
	return newDiscovery(c, h, func(maddr ma.Multiaddr) (net.PacketConn, error) {
		network, _, err := manet.DialArgs(maddr)
		if err != nil {
			return nil, err
		}

		return net.ListenPacket(network, ":0")
	})
}

func Listen(c *cli.Context, h host.Host) (DiscoveryService, error) {
	return newDiscovery(c, h, func(maddr ma.Multiaddr) (net.PacketConn, error) {
		network, address, err := manet.DialArgs(maddr)
		if err != nil {
			return nil, err
		}

		_, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}

		return net.ListenPacket(network, ":"+port)
	})
}

func newDiscovery(c *cli.Context, h host.Host, newConn func(ma.Multiaddr) (net.PacketConn, error)) (DiscoveryService, error) {
	log := logutil.New(c)

	maddr, err := ma.NewMultiaddr(c.String("discover"))
	if err != nil {
		return nil, err
	}

	switch {
	case crawler(maddr):
		s, err := crawl.ParseCIDR(maddr)
		if err != nil {
			return nil, err
		}

		conn, err := newConn(maddr)
		if err != nil {
			return nil, err
		}

		return crawl.New(h, conn, s, socket.WithLogger(log)), nil

	case multicast(maddr):
		group, ifi, err := survey.ResolveMulticast(maddr)
		if err != nil {
			return nil, err
		}

		conn, err := survey.JoinMulticastGroup("udp", ifi, group)
		if err != nil {
			return nil, err
		}

		s := survey.New(h, conn, socket.WithLogger(log))

		if !gradual(maddr) {
			return s, nil
		}

		return survey.GradualSurveyor{Surveyor: s}, nil
	}

	return nil, ErrUnknownBootProto
}

func crawler(maddr ma.Multiaddr) bool {
	return hasBootProto(maddr, crawl.P_CIDR)
}

func multicast(maddr ma.Multiaddr) bool {
	return hasBootProto(maddr, survey.P_MULTICAST)
}

func gradual(maddr ma.Multiaddr) bool {
	return hasBootProto(maddr, survey.P_SURVEY)
}

func hasBootProto(maddr ma.Multiaddr, code int) bool {
	for _, p := range maddr.Protocols() {
		if p.Code == code {
			return true
		}
	}

	return false
}
