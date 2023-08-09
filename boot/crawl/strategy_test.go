package crawl_test

import (
	"net"
	"testing"

	ma "github.com/multiformats/go-multiaddr"

	"github.com/wetware/ww/boot/crawl"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPortRange(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("Reset", func(t *testing.T) {
		t.Parallel()

		var pr crawl.PortRange
		pr.Reset()

		assert.Equal(t, net.IPv4(127, 0, 0, 1), pr.IP, "should default to loopback IP")
		assert.Equal(t, uint16(1024), pr.Low, "should start at port 1024 by default")
		assert.Equal(t, uint16(65535), pr.High, "should stop at port 65535 by default")
	})

	t.Run("DefaultRange", func(t *testing.T) {
		t.Parallel()

		pr, err := crawl.NewPortRange(nil, 0, 0)()
		require.NoError(t, err)
		require.NotNil(t, pr)

		seen := map[int]struct{}{}

		var addr net.UDPAddr
		for pr.Next(&addr) {
			seen[addr.Port] = struct{}{}
		}

		assert.Len(t, seen, 64512, "should contain all non-reserved ports")

		for i := 0; i < 1024; i++ {
			require.NotContains(t, seen, i,
				"should not contain reserved port %d", i)
		}
	})

	t.Run("Reserved", func(t *testing.T) {
		t.Parallel()

		pr, err := crawl.NewPortRange(nil, 1, 1024)()
		require.NoError(t, err)
		require.NotNil(t, pr)

		seen := map[int]struct{}{}

		var addr net.UDPAddr
		for pr.Next(&addr) {
			seen[addr.Port] = struct{}{}
		}

		assert.Len(t, seen, 1024, "should contain all reserved ports")
	})

	t.Run("Single", func(t *testing.T) {
		t.Parallel()

		pr, err := crawl.NewPortRange(nil, 8822, 8822)()
		require.NoError(t, err)
		require.NotNil(t, pr)

		seen := map[int]struct{}{}

		var addr net.UDPAddr
		for pr.Next(&addr) {
			seen[addr.Port] = struct{}{}
		}

		assert.Len(t, seen, 1, "should contain exactly one ports")
		assert.Contains(t, seen, 8822, "should contain specified port")
	})
}

func TestCIDR(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("IPv4", func(t *testing.T) {
		t.Parallel()
		t.Helper()

		t.Run("ByteAligned", func(t *testing.T) {
			t.Parallel()

			maddr := ma.StringCast("/ip4/228.8.8.8/udp/8822/cidr/24")

			cidr, err := crawl.ParseCIDR(maddr)
			require.NoError(t, err, "should succeed")
			require.NotNil(t, cidr, "should return strategy")

			c, err := cidr()
			assert.NoError(t, err, "should succeed")
			assert.IsType(t, new(crawl.CIDR), c, "should return CIDR range")

			seen := map[string]struct{}{}

			var addr net.UDPAddr
			for c.Next(&addr) {
				seen[addr.String()] = struct{}{}
			}

			assert.Len(t, seen, 254,
				"should contain 8-bit subnet without network & broadcast addrs")
		})

		t.Run("Unaligned", func(t *testing.T) {
			t.Parallel()

			maddr := ma.StringCast("/ip4/228.8.8.8/udp/8822/cidr/21")

			cidr, err := crawl.ParseCIDR(maddr)
			require.NoError(t, err, "should succeed")
			require.NotNil(t, cidr, "should return strategy")

			c, err := cidr()
			assert.NoError(t, err, "should succeed")
			assert.IsType(t, new(crawl.CIDR), c, "should return CIDR range")

			seen := map[string]struct{}{}

			var addr net.UDPAddr
			for c.Next(&addr) {
				seen[addr.String()] = struct{}{}
			}

			assert.Len(t, seen, 2046,
				"should contain 6-bit subnet without network & broadcast addrs")
		})
	})

	t.Run("IPv6", func(t *testing.T) {
		t.Parallel()
		t.Helper()

		t.Run("ByteAligned", func(t *testing.T) {
			t.Parallel()

			maddr := ma.StringCast("/ip6/2001:db8::/udp/8822/cidr/120")

			cidr, err := crawl.ParseCIDR(maddr)
			require.NoError(t, err, "should succeed")
			require.NotNil(t, cidr, "should return strategy")

			c, err := cidr()
			assert.NoError(t, err, "should succeed")
			assert.IsType(t, new(crawl.CIDR), c, "should return CIDR range")

			seen := map[string]struct{}{}

			var addr net.UDPAddr
			for c.Next(&addr) {
				seen[addr.String()] = struct{}{}
			}

			assert.Len(t, seen, 254,
				"should contain 8-bit subnet without network & broadcast addrs")
		})

		t.Run("Unaligned", func(t *testing.T) {
			t.Parallel()

			maddr := ma.StringCast("/ip6/2001:db8::/udp/8822/cidr/117")

			cidr, err := crawl.ParseCIDR(maddr)
			require.NoError(t, err, "should succeed")
			require.NotNil(t, cidr, "should return strategy")

			c, err := cidr()
			assert.NoError(t, err, "should succeed")
			assert.IsType(t, new(crawl.CIDR), c, "should return CIDR range")

			seen := map[string]struct{}{}

			var addr net.UDPAddr
			for c.Next(&addr) {
				seen[addr.String()] = struct{}{}
			}

			t.Log(len(seen))
			assert.Len(t, seen, 2046,
				"should contain 6-bit subnet without network & broadcast addrs")
		})
	})
}

func BenchmarkCIDR(b *testing.B) {
	for _, bt := range []struct {
		name string
		CIDR ma.Multiaddr
	}{
		{
			name: "24",
			CIDR: ma.StringCast("/ip4/228.8.8.8/udp/8822/cidr/24"),
		},
		{
			name: "16",
			CIDR: ma.StringCast("/ip4/228.8.8.8/udp/8822/cidr/16"),
		},
		{
			name: "8",
			CIDR: ma.StringCast("/ip4/228.8.8.8/udp/8822/cidr/8"),
		},
	} {
		b.Run(bt.name, func(b *testing.B) {
			cidr, err := crawl.ParseCIDR(bt.CIDR)
			if err != nil {
				panic(err)
			}

			c, err := cidr()
			if err != nil {
				panic(err)
			}

			b.ResetTimer()

			var addr net.UDPAddr
			for i := 0; i < b.N; i++ {
				for c.Next(&addr) {
					// iterate...
				}
			}
		})
	}
}
