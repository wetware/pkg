// Package boot provides utilities for parsing and instantiating boot services
package boot

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"

	"github.com/wetware/pkg/boot/crawl"
	"github.com/wetware/pkg/boot/socket"
	"github.com/wetware/pkg/boot/survey"
)

// ErrUnknownBootProto is returned when the multiaddr passed
// to Parse does not contain a recognized boot protocol.
var ErrUnknownBootProto = errors.New("unknown boot protocol")

func DialString(h host.Host, s string, opt ...socket.Option) (discovery.Discoverer, error) {
	maddr, err := ma.NewMultiaddr(s)
	if err != nil {
		return nil, err
	}

	return Dial(h, maddr, opt...)
}

func Dial(h host.Host, maddr ma.Multiaddr, opt ...socket.Option) (discovery.Discoverer, error) {
	switch {
	case IsP2P(maddr):
		return NewStaticAddrs(maddr)

	case IsCIDR(maddr):
		return DialCIDR(h, maddr, opt...)

	case IsMulticast(maddr):
		s, err := DialMulticast(h, maddr, opt...)
		if err != nil {
			return nil, err
		}

		if IsSurvey(maddr) {
			return survey.GradualSurveyor{Surveyor: s}, nil
		}

		return s, nil

	case IsPortRange(maddr):
		return DialPortRange(h, maddr, opt...)
	}

	return nil, ErrUnknownBootProto
}

func ListenString(h host.Host, s string, opt ...socket.Option) (discovery.Discovery, error) {
	maddr, err := ma.NewMultiaddr(s)
	if err != nil {
		return nil, err
	}

	return Listen(h, maddr, opt...)
}

func Listen(h host.Host, maddr ma.Multiaddr, opt ...socket.Option) (discovery.Discovery, error) {
	switch {
	case IsCIDR(maddr):
		return ListenCIDR(h, maddr, opt...)

	case IsMulticast(maddr):
		s, err := DialMulticast(h, maddr, opt...)
		if err != nil {
			return nil, err
		}

		if IsSurvey(maddr) {
			return survey.GradualSurveyor{Surveyor: s}, nil
		}

		return s, nil
	}

	return nil, ErrUnknownBootProto
}

func DialCIDR(h host.Host, maddr ma.Multiaddr, opt ...socket.Option) (*crawl.Crawler, error) {
	return newCIDRCrawler(h, maddr, dial, opt)
}

func ListenCIDR(h host.Host, maddr ma.Multiaddr, opt ...socket.Option) (*crawl.Crawler, error) {
	return newCIDRCrawler(h, maddr, listen, opt)
}

func DialMulticast(h host.Host, maddr ma.Multiaddr, opt ...socket.Option) (*survey.Surveyor, error) {
	group, ifi, err := survey.ResolveMulticast(maddr)
	if err != nil {
		return nil, err
	}

	conn, err := survey.JoinMulticastGroup("udp", ifi, group)
	if err != nil {
		return nil, err
	}

	return survey.New(h, conn, opt...), nil
}

func DialPortRange(h host.Host, maddr ma.Multiaddr, opt ...socket.Option) (*crawl.Crawler, error) {
	_, addr, err := manet.DialArgs(maddr)
	if err != nil {
		return nil, err
	}

	ipstr, portstr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	ip := net.ParseIP(ipstr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP: %s", ipstr)
	}

	port, err := strconv.Atoi(portstr)
	if err != nil {
		return nil, err
	}

	return crawlerFactory{
		Host:     h,
		Strategy: crawl.NewPortRange(ip, port, port),
		NewConn:  dial,
	}.New(maddr, opt)
}

func IsP2P(maddr ma.Multiaddr) bool {
	return hasBootProto(maddr, ma.P_P2P)
}

func IsCIDR(maddr ma.Multiaddr) bool {
	return hasBootProto(maddr, crawl.P_CIDR)
}

func IsMulticast(maddr ma.Multiaddr) bool {
	return hasBootProto(maddr, survey.P_MULTICAST)
}

func IsSurvey(maddr ma.Multiaddr) bool {
	return hasBootProto(maddr, survey.P_SURVEY)
}

// IsPortRange returns true if maddr is a UDP address with no subprotocols.
// This function MAY be extended to support port ranges ranges in the near
// future.
func IsPortRange(maddr ma.Multiaddr) bool {
	var n int
	ma.ForEach(maddr, func(ma.Component) bool {
		n++
		return true
	})

	// are there more than two components?
	if n > 2 {
		return false
	}

	// are the components not ip & udp?
	hasIP := hasBootProto(maddr, ma.P_IP4) || hasBootProto(maddr, ma.P_IP6)
	hasUDP := hasBootProto(maddr, ma.P_UDP)
	return hasIP && hasUDP
}

func hasBootProto(maddr ma.Multiaddr, code int) bool {
	for _, p := range maddr.Protocols() {
		if p.Code == code {
			return true
		}
	}

	return false
}

func dial(maddr ma.Multiaddr) (net.PacketConn, error) {
	network, _, err := manet.DialArgs(maddr)
	if err != nil {
		return nil, err
	}

	return net.ListenPacket(network, ":0")
}

func listen(maddr ma.Multiaddr) (net.PacketConn, error) {
	network, address, err := manet.DialArgs(maddr)
	if err != nil {
		return nil, err
	}

	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	return net.ListenPacket(network, ":"+port)
}

type packetFunc func(ma.Multiaddr) (net.PacketConn, error)

func newCIDRCrawler(h host.Host, maddr ma.Multiaddr, newConn packetFunc, opt []socket.Option) (*crawl.Crawler, error) {
	s, err := crawl.ParseCIDR(maddr)
	if err != nil {
		return nil, err
	}

	return crawlerFactory{
		Host:     h,
		Strategy: s,
		NewConn:  newConn,
	}.New(maddr, opt)
}

type crawlerFactory struct {
	Host     host.Host
	Strategy crawl.Strategy
	NewConn  packetFunc
}

func (c crawlerFactory) New(addr ma.Multiaddr, opt []socket.Option) (*crawl.Crawler, error) {
	conn, err := c.NewConn(addr)
	if err != nil {
		return nil, err
	}

	return crawl.New(c.Host, conn, c.Strategy, opt...), nil
}
