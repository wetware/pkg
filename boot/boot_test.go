package boot_test

import (
	"testing"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/wetware/ww/boot"
)

func TestIsCIDR(t *testing.T) {
	t.Parallel()
	t.Helper()

	for _, tt := range []struct {
		name string
		addr ma.Multiaddr
		want bool
	}{
		{name: "IPv4", addr: ma.StringCast("/ip4/10.0.1.0/udp/8822/cidr/24"), want: true},
		{name: "IPv6", addr: ma.StringCast("/ip6/2001:ff::/udp/8822/cidr/110"), want: true},
		{name: "Fail", addr: ma.StringCast("/ip6/2001:ff::/udp/8822"), want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, boot.IsCIDR(tt.addr))
		})
	}
}

func TestIsMulticast(t *testing.T) {
	t.Parallel()
	t.Helper()

	for _, tt := range []struct {
		name string
		addr ma.Multiaddr
		want bool
	}{
		{name: "IPv4", addr: ma.StringCast("/ip4/10.0.1.0/udp/8822/multicast/lo0"), want: true},
		{name: "IPv6", addr: ma.StringCast("/ip6/2001:ff::/udp/8822/multicast/lo0"), want: true},
		{name: "IPv4/Survey", addr: ma.StringCast("/ip4/10.0.1.0/udp/8822/multicast/lo0/survey"), want: true},
		{name: "IPv6/Survey", addr: ma.StringCast("/ip6/2001:ff::/udp/8822/multicast/lo0/survey"), want: true},
		{name: "Fail", addr: ma.StringCast("/ip6/2001:ff::/udp/8822"), want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, boot.IsMulticast(tt.addr))
		})
	}
}

func TestIsSurvey(t *testing.T) {
	t.Parallel()
	t.Helper()

	for _, tt := range []struct {
		name string
		addr ma.Multiaddr
		want bool
	}{
		{name: "IPv4", addr: ma.StringCast("/ip4/10.0.1.0/udp/8822/multicast/lo0/survey"), want: true},
		{name: "IPv6", addr: ma.StringCast("/ip6/2001:ff::/udp/8822/multicast/lo0/survey"), want: true},
		{name: "Fail", addr: ma.StringCast("/ip6/2001:ff::/udp/8822/multicast/lo0"), want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, boot.IsSurvey(tt.addr))
		})
	}
}

func TestIsPortRange(t *testing.T) {
	t.Parallel()
	t.Helper()

	for _, tt := range []struct {
		name string
		addr ma.Multiaddr
		want bool
	}{
		{name: "IPv4", addr: ma.StringCast("/ip4/10.0.1.0/udp/8822"), want: true},
		{name: "IPv6", addr: ma.StringCast("/ip6/2001:ff::/udp/8822"), want: true},
		{name: "TCP", addr: ma.StringCast("/ip4/10.0.1.0/tcp/8822"), want: false},
		{name: "Subprotocol", addr: ma.StringCast("/ip4/10.0.1.0/udp/8822/quic"), want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, boot.IsPortRange(tt.addr))
		})
	}
}
