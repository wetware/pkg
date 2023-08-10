package crawl_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/record"
	inproc "github.com/lthibault/go-libp2p-inproc-transport"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wetware/pkg/boot/crawl"
	"github.com/wetware/pkg/boot/socket"
	mock_net "github.com/wetware/pkg/test/net"
)

const (
	reqfile = "../socket/testdata/request.golden.capnp"
	resfile = "../socket/testdata/response.golden.capnp"
)

var reqBytes, resBytes []byte

func init() {
	var err error
	if reqBytes, err = os.ReadFile(reqfile); err != nil {
		panic(err)
	}

	if resBytes, err = os.ReadFile(resfile); err != nil {
		panic(err)
	}
}

func TestMultiaddr(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		addr string
		fail bool
	}{
		{"/ip4/228.8.8.8/udp/8822/cidr/32", false},
		{"/ip4/228.8.8.8/udp/8822/cidr/129", true},
	} {
		_, err := ma.NewMultiaddr(tt.addr)
		if tt.fail {
			assert.Error(t, err, "should fail to parse %s", tt.addr)
		} else {
			assert.NoError(t, err, "should parse %s", tt.addr)
		}
	}
}

func TestTranscoderCIDR(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("StringToBytes", func(t *testing.T) {
		t.Parallel()

		s, err := crawl.TranscoderCIDR{}.BytesToString([]byte{0x00, 0x00})
		assert.ErrorIs(t, err, crawl.ErrCIDROverflow,
			"should not parse byte arrays of length > 1")
		assert.Empty(t, s)

		s, err = crawl.TranscoderCIDR{}.BytesToString([]byte{0xFF})
		assert.ErrorIs(t, err, crawl.ErrCIDROverflow,
			"should not validate CIDR greater than 128")
		assert.Empty(t, s)

		s, err = crawl.TranscoderCIDR{}.BytesToString([]byte{0x01})
		assert.NoError(t, err, "should parse CIDR of 1")
		assert.Equal(t, "1", s, "should return \"1\"")
	})

	t.Run("BytesToString", func(t *testing.T) {
		t.Parallel()

		b, err := crawl.TranscoderCIDR{}.StringToBytes("fail")
		assert.Error(t, err,
			"should not validate non-numerical strings")
		assert.Nil(t, b)

		b, err = crawl.TranscoderCIDR{}.StringToBytes("255")
		assert.ErrorIs(t, err, crawl.ErrCIDROverflow,
			"should not validate string '255'")
		assert.Nil(t, b)
	})

	t.Run("ValidateBytes", func(t *testing.T) {
		t.Parallel()

		err := crawl.TranscoderCIDR{}.ValidateBytes([]byte{0x00})
		assert.NoError(t, err,
			"should validate CIDR block of 0")

		err = crawl.TranscoderCIDR{}.ValidateBytes([]byte{0xFF})
		assert.ErrorIs(t, err, crawl.ErrCIDROverflow,
			"should not validate CIDR blocks greater than 128")
	})
}

func TestCrawler_request_noadvert(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sync := make(chan struct{})
	addr := &net.UDPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 8822,
	}

	h := newTestHost()
	defer h.Close()

	// logger := logtest.NewMockLogger(ctrl)

	conn := mock_net.NewMockPacketConn(ctrl)
	conn.EXPECT().
		Close().
		DoAndReturn(func() error {
			defer close(sync)
			return nil
		}).
		Times(1)

	readReq := conn.EXPECT().
		ReadFrom(gomock.Any()).
		DoAndReturn(func(b []byte) (n int, a net.Addr, err error) {
			n = copy(b, reqBytes)
			a = addr
			return
		}).
		Times(1)

	conn.EXPECT().
		ReadFrom(gomock.Any()).
		After(readReq).
		DoAndReturn(blockUntilClosed(sync)).
		AnyTimes()

	c := crawl.New(h, conn, rangeUDP(),
		/*socket.WithLogger(logger)*/)
	assert.NoError(t, c.Close(), "should close gracefully")
}

func TestCrawler_advertise(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		syncClose  = make(chan struct{})
		syncAdvert = make(chan struct{})
		syncReply  = make(chan struct{})
	)

	addr := &net.UDPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 8822,
	}

	h := newTestHost()
	defer h.Close()

	// logger := logtest.NewMockLogger(ctrl)

	conn := mock_net.NewMockPacketConn(ctrl)
	conn.EXPECT().
		Close().
		DoAndReturn(func() error {
			defer close(syncClose)
			return nil
		}).
		Times(1)

	// expect an incoming request
	readReq := conn.EXPECT().
		ReadFrom(gomock.Any()).
		DoAndReturn(func(b []byte) (n int, a net.Addr, err error) {
			n = copy(b, reqBytes)
			a = addr
			<-syncAdvert
			return
		}).
		Times(1)

	// expect to write a response
	conn.EXPECT().
		WriteTo(matchOutgoingResponse(), gomock.Eq(addr)).
		After(readReq).
		DoAndReturn(func(b []byte, _ net.Addr) (int, error) {
			defer close(syncReply)
			return len(b), nil
		}).
		Times(1)

	// block on next read indefinitely
	conn.EXPECT().
		ReadFrom(gomock.Any()).
		After(readReq).
		Return(0, nil, net.ErrClosed).
		DoAndReturn(blockUntilClosed(syncClose)).
		AnyTimes()

	c := crawl.New(h, conn, rangeUDP(),
		/*socket.WithLogger(logger)*/)
	defer c.Close()

	ttl, err := c.Advertise(ctx, "casm")
	require.NoError(t, err, "advertise should succeed")
	assert.Equal(t, peerstore.TempAddrTTL, ttl)
	close(syncAdvert)

	<-syncReply
}

func TestCrawler_FindPeers_strategy_error(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := newTestHost()
	defer h.Close()

	// logger := logtest.NewMockLogger(ctrl)

	conn := mock_net.NewMockPacketConn(ctrl)
	conn.EXPECT().
		ReadFrom(gomock.Any()).
		DoAndReturn(func(b []byte) (n int, a net.Addr, err error) {
			n = copy(b, resBytes)
			a = &net.UDPAddr{}
			return
		}).
		AnyTimes()
	conn.EXPECT().
		Close().
		Return(nil).
		Times(1)

	errFail := errors.New("fail")
	fail := func() (crawl.Range, error) {
		return nil, errFail
	}

	c := crawl.New(h, conn, fail, /*socket.WithLogger(logger)*/)
	defer func() {
		assert.NoError(t, c.Close(), "should close gracefully")
	}()

	ch, err := c.FindPeers(context.TODO(), "test")
	require.ErrorIs(t, err, errFail, "should return strategy error")
	require.Nil(t, ch, "should return nil channel")
}

func TestCrawler_FindPeers_wait(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("GracefulAbort", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		h := newTestHost()
		defer h.Close()

		// logger := logtest.NewMockLogger(ctrl)
		// logger.EXPECT().
		// 	WithField(gomock.Any(), gomock.Any()).
		// 	Return(logger).
		// 	AnyTimes()
		// logger.EXPECT().
		// 	Trace(gomock.Any()).
		// 	AnyTimes()

		conn := mock_net.NewMockPacketConn(ctrl)
		conn.EXPECT().
			ReadFrom(gomock.Any()).
			DoAndReturn(func(b []byte) (n int, a net.Addr, err error) {
				n = copy(b, resBytes)
				a = &net.UDPAddr{}
				return
			}).
			AnyTimes()
		conn.EXPECT().
			Close().
			Return(nil).
			Times(1)
		conn.EXPECT().
			WriteTo(gomock.Any(), gomock.Any()).
			Return(0, context.Canceled).
			Times(1)

		c := crawl.New(h, conn, rangeUDP(&net.UDPAddr{}), /*socket.WithLogger(logger)*/)
		defer c.Close()

		ch, err := c.FindPeers(context.TODO(), "test")
		require.NoError(t, err, "should not return error")
		require.NotNil(t, ch, "should return valid channel")

		assert.Eventually(t, func() bool {
			select {
			case <-ch:
				return true
			default:
				return false
			}
		}, time.Millisecond*100, time.Millisecond*10,
			"should close channel when error is encountered")
	})

	t.Run("SocketError", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		h := newTestHost()
		defer h.Close()

		errFail := errors.New("fail")

		// logger := logtest.NewMockLogger(ctrl)
		// logger.EXPECT().
		// 	WithField(gomock.Any(), gomock.Any()).
		// 	Return(logger).
		// 	AnyTimes()
		// logger.EXPECT().
		// 	WithError(errFail).
		// 	Return(logger).
		// 	Times(1)
		// logger.EXPECT().
		// 	Error("failed to send request packet").
		// 	Times(1)

		conn := mock_net.NewMockPacketConn(ctrl)
		conn.EXPECT().
			ReadFrom(gomock.Any()).
			DoAndReturn(func(b []byte) (n int, a net.Addr, err error) {
				n = copy(b, resBytes)
				a = &net.UDPAddr{}
				return
			}).
			AnyTimes()
		conn.EXPECT().
			Close().
			Return(nil).
			Times(1)
		conn.EXPECT().
			WriteTo(gomock.Any(), gomock.Any()).
			Return(0, errFail).
			Times(1)

		c := crawl.New(h, conn, rangeUDP(&net.UDPAddr{}), /*socket.WithLogger(logger)*/)
		defer c.Close()

		ch, err := c.FindPeers(context.TODO(), "test")
		require.NoError(t, err, "should not return error")
		require.NotNil(t, ch, "should return valid channel")

		assert.Eventually(t, func() bool {
			select {
			case <-ch:
				return true
			default:
				return false
			}
		}, time.Millisecond*100, time.Millisecond*10,
			"should close channel when error is encountered")
	})
}

func TestCrawler_find_peers(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		syncClose = make(chan struct{})
		syncReq   = make(chan struct{})
	)

	addr := &net.UDPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 8822,
	}

	h := newTestHost()
	defer h.Close()

	// logger := logtest.NewMockLogger(ctrl)
	// logger.EXPECT().
	// 	WithField(gomock.Any(), gomock.Any()).
	// 	Return(logger).
	// 	AnyTimes()
	// logger.EXPECT().
	// 	Trace(gomock.Any()).
	// 	AnyTimes()

	conn := mock_net.NewMockPacketConn(ctrl)
	conn.EXPECT().
		Close().
		DoAndReturn(func() error {
			defer close(syncClose)
			return nil
		}).
		Times(1)

	conn.EXPECT().
		WriteTo(gomock.Any(), gomock.Eq(addr)).
		DoAndReturn(func(b []byte, _ net.Addr) (int, error) {
			defer close(syncReq)
			return len(b), nil
		}).
		Times(1)

	conn.EXPECT().
		ReadFrom(gomock.Any()).
		DoAndReturn(func(b []byte) (n int, a net.Addr, err error) {
			n = copy(b, resBytes)
			a = addr
			<-syncReq
			return
		}).
		MinTimes(1)

	conn.EXPECT().
		ReadFrom(gomock.Any()).
		DoAndReturn(blockUntilClosed(syncClose)).
		AnyTimes()

	c := crawl.New(h, conn, rangeUDP(addr),
		/*socket.WithLogger(logger)*/)
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	peers, err := c.FindPeers(ctx, "casm")
	require.NoError(t, err, "should not return error")

	var ps []peer.AddrInfo
	for info := range peers {
		ps = append(ps, info)
		break
	}

	assert.NotEmpty(t, ps, "should find one peer")
}

func blockUntilClosed(sync <-chan struct{}) func([]byte) (int, net.Addr, error) {
	return func([]byte) (int, net.Addr, error) {
		<-sync
		return 0, nil, net.ErrClosed
	}
}

func matchOutgoingResponse() gomock.Matcher {
	return &matchResponse{}
}

type matchResponse struct {
	err error
}

// Matches returns whether x is a match.
func (m *matchResponse) Matches(x interface{}) bool {
	b, ok := x.([]byte)
	if !ok {
		m.err = fmt.Errorf("expected *boot.Record, got %s", reflect.TypeOf(x))
		return false
	}

	_, rec, err := record.ConsumeEnvelope(b, socket.EnvelopeDomain)
	if err != nil {
		m.err = fmt.Errorf("consume envelope: %w", err)
		return false
	}

	if _, ok = rec.(*socket.Record); !ok {
		m.err = fmt.Errorf("expected *boot.Record, got %s", reflect.TypeOf(rec))
		return false
	}

	return true
}

// String describes what the matcher matches.
func (m matchResponse) String() string {
	if m.err != nil {
		return m.err.Error()
	}

	return "is response packet"
}

type mockRange struct {
	pos int
	as  []*net.UDPAddr
}

func rangeUDP(as ...*net.UDPAddr) crawl.Strategy {
	return func() (crawl.Range, error) {
		return &mockRange{as: as}, nil
	}
}

func (r *mockRange) Next(a net.Addr) bool {
	if r.pos == len(r.as) {
		return false
	}

	switch addr := a.(type) {
	case *net.UDPAddr:
		addr.IP = r.as[r.pos].IP
		addr.Zone = r.as[r.pos].Zone
		addr.Port = r.as[r.pos].Port
		r.pos++
		return true
	}

	panic("unreachable")
}

func newTestHost() host.Host {
	h, err := libp2p.New(
		libp2p.NoListenAddrs,
		libp2p.NoTransports,
		libp2p.Transport(inproc.New()),
		libp2p.ListenAddrStrings("/inproc/~"))
	if err != nil {
		panic(err)
	}

	return h
}
