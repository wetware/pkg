package survey

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

const (
	P_MULTICAST = iota + 100
	P_SURVEY
)

func init() {
	for _, p := range []ma.Protocol{
		{
			Name:       "multicast",
			Code:       P_MULTICAST,
			VCode:      ma.CodeToVarint(P_MULTICAST),
			Size:       ma.LengthPrefixedVarSize,
			Transcoder: TranscoderIface{},
		},
		{
			Name:  "survey",
			Code:  P_SURVEY,
			VCode: ma.CodeToVarint(P_SURVEY),
		},
	} {
		if err := ma.AddProtocol(p); err != nil {
			panic(err)
		}
	}
}

func ResolveMulticast(maddr ma.Multiaddr) (*net.UDPAddr, *net.Interface, error) {
	network, addr, err := manet.DialArgs(maddr)
	if err != nil {
		return nil, nil, err
	}

	udp, err := net.ResolveUDPAddr(network, addr)
	if err != nil {
		return nil, nil, err
	}

	ifi, err := ResolveMulticastInterface(maddr)
	return udp, ifi, err
}

func ResolveMulticastInterface(maddr ma.Multiaddr) (*net.Interface, error) {
	name, err := maddr.ValueForProtocol(P_MULTICAST)
	if err != nil {
		return nil, err
	}

	ifi, err := net.InterfaceByName(name)
	if err != nil {
		return nil, err
	}

	if ifi.Flags&net.FlagUp == 0 {
		return nil, fmt.Errorf("%s: interface down", name)
	}

	if ifi.Flags&net.FlagMulticast == 0 {
		return nil, fmt.Errorf("%s: multicast disabled", name)
	}

	if strings.HasPrefix(name, "lo") && ifi.Flags&net.FlagLoopback == 0 {
		return nil, fmt.Errorf("%s: looback disabled", name)
	}

	return ifi, nil
}

// JoinMulticastGroup joins the group address group on the provided
// interface. By default all sources that can cast data to group are
// accepted.
//
// If ifi == nil, JoinMulticastGroup uses the system-assigned multicast
// interface.  Note that users SHOULD NOT do this because the resulting
// interface is platform-dependent, and may require special routing config.
func JoinMulticastGroup(network string, ifi *net.Interface, group *net.UDPAddr) (net.PacketConn, error) {
	if !group.IP.IsMulticast() {
		return nil, errors.New("not a multicast addr")
	}

	c, err := net.ListenUDP(network, group)
	if err != nil {
		return nil, err
	}

	if group.IP.To4() != nil {
		return newIPv4MulticastConn(c, ifi, group)
	}

	return newIPv6MulticastConn(c, ifi, group)
}

type ipv4MulticastConn struct {
	*ipv4.PacketConn
	group *net.UDPAddr
}

func newIPv4MulticastConn(c net.PacketConn, ifi *net.Interface, group *net.UDPAddr) (net.PacketConn, error) {
	conn := ipv4.NewPacketConn(c)

	err := conn.JoinGroup(ifi, &net.UDPAddr{IP: group.IP})
	if err != nil {
		return nil, err
	}

	if err = conn.SetControlMessage(ipv4.FlagDst, true); err != nil {
		return nil, err
	}

	if err := conn.SetMulticastInterface(ifi); err != nil {
		return nil, err
	}

	return ipv4MulticastConn{
		PacketConn: conn,
		group:      group,
	}, nil
}

// LocalAddr returns the multicast group to which the connnection belongs.
// The returned net.Addr is shared across calls, and MUST NOT be modified.
func (c ipv4MulticastConn) LocalAddr() net.Addr { return c.group }

// returns unicast address of originator.
func (c ipv4MulticastConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	var cm *ipv4.ControlMessage
	for {
		if n, cm, addr, err = c.PacketConn.ReadFrom(b); err != nil {
			return
		}

		if cm.Dst.IsMulticast() {
			return
		}
	}
}

func (c ipv4MulticastConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	return c.PacketConn.WriteTo(b, nil, addr)
}

type ipv6MulticastConn struct {
	*ipv6.PacketConn
	group *net.UDPAddr
}

func newIPv6MulticastConn(c net.PacketConn, ifi *net.Interface, group *net.UDPAddr) (net.PacketConn, error) {
	conn := ipv6.NewPacketConn(c)

	err := conn.JoinGroup(ifi, &net.UDPAddr{IP: group.IP})
	if err != nil {
		return nil, err
	}

	if err = conn.SetControlMessage(ipv6.FlagDst, true); err != nil {
		return nil, err
	}

	if err = conn.SetMulticastInterface(ifi); err != nil {
		return nil, err
	}

	return ipv6MulticastConn{
		PacketConn: conn,
		group:      group,
	}, err
}

// LocalAddr returns the multicast group to which the connnection belongs
// The returned net.Addr is shared across calls, and MUST NOT be modified.
func (c ipv6MulticastConn) LocalAddr() net.Addr { return c.group }

func (c ipv6MulticastConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	var cm *ipv6.ControlMessage
	for {
		if n, cm, addr, err = c.PacketConn.ReadFrom(b); err != nil {
			return
		}

		if cm.Dst.IsMulticast() {
			return
		}
	}
}

func (c ipv6MulticastConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	return c.PacketConn.WriteTo(b, nil, addr)
}

// TranscoderIface decodes an interface name.
type TranscoderIface struct{}

func (it TranscoderIface) StringToBytes(name string) ([]byte, error) {
	return []byte(name), nil
}

func (it TranscoderIface) BytesToString(b []byte) (string, error) {
	return string(b), nil
}

func (it TranscoderIface) ValidateBytes(b []byte) error { return nil }
