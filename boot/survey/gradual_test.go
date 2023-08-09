package survey_test

// import (
// 	"context"
// 	"net"
// 	"testing"
// 	"time"

// 	"capnproto.org/go/capnp/v3"
// 	"github.com/golang/mock/gomock"
// 	"github.com/libp2p/go-libp2p/core/discovery"
// 	"github.com/libp2p/go-libp2p/core/event"
// 	"github.com/libp2p/go-libp2p/core/host"
// 	"github.com/stretchr/testify/require"
// 	"github.com/wetware/ww/api/boot"
// 	mock_net "github.com/wetware/casm/internal/mock/net"
// 	"github.com/wetware/casm/pkg/boot/survey"
// )

// func TestDiscoverGradual(t *testing.T) {
// 	t.Parallel()

// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	h := newTestHost()
// 	defer h.Close()

// 	sub, err := h.EventBus().Subscribe(new(event.EvtLocalAddressesUpdated))
// 	require.NoError(t, err, "must subscribe to address updates")
// 	defer sub.Close()
// 	e := (<-sub.Out()).(event.EvtLocalAddressesUpdated).SignedPeerRecord

// 	resp := make(chan []byte, 1)
// 	mockTransport := survey.MulticastTransport{
// 		DialFunc: func(net.Addr) (net.PacketConn, error) {
// 			conn := mock_net.NewMockPacketConn(ctrl)
// 			// Expect a single call to Close
// 			conn.EXPECT().
// 				Close().
// 				Return(error(nil)).
// 				Times(1)

// 			// Expect a multiple REQUEST packet to be issued with incrementing distance.
// 			conn.EXPECT().
// 				WriteTo(gomock.AssignableToTypeOf([]byte{}), gomock.AssignableToTypeOf(net.Addr(new(net.UDPAddr)))).
// 				DoAndReturn(func(b []byte, _ net.Addr) (int, error) {
// 					m, err := capnp.UnmarshalPacked(b)
// 					if err != nil {
// 						return 0, err
// 					}

// 					p, err := boot.ReadRootPacket(m)
// 					if err != nil {
// 						return 0, err
// 					}

// 					r, err := p.Request()
// 					if err != nil {
// 						return 0, err
// 					}

// 					if r.Distance() > 3 {
// 						select {
// 						case resp <- newResponsePayload(e):
// 						default:
// 						}
// 					}

// 					return len(b), nil
// 				}).
// 				AnyTimes()

// 			return conn, nil
// 		},
// 		ListenFunc: func(net.Addr) (net.PacketConn, error) {
// 			conn := mock_net.NewMockPacketConn(ctrl)
// 			// Expect a single call to Close
// 			conn.EXPECT().
// 				Close().
// 				Return(error(nil)).
// 				Times(1)

// 				// Expect one RESPONSE message.
// 			conn.EXPECT().
// 				ReadFrom(gomock.AssignableToTypeOf([]byte{})).
// 				DoAndReturn(func(b []byte) (int, net.Addr, error) {
// 					select {
// 					case raw := <-resp:
// 						return copy(b, raw), new(net.UDPAddr), nil
// 					case <-ctx.Done():
// 						return 0, nil, ctx.Err()
// 					}
// 				}).
// 				AnyTimes()

// 			return conn, nil
// 		},
// 	}

// 	s, err := survey.New(h, new(net.UDPAddr), survey.WithTransport(mockTransport))
// 	require.NoError(t, err, "should open packet connections")
// 	require.NotNil(t, s, "should return surveyor")
// 	defer s.Close()

// 	g := survey.GradualSurveyor{
// 		Surveyor: s,
// 		Min:      time.Millisecond,
// 		Max:      time.Millisecond * 10,
// 	}

// 	// Advertise ...
// 	ttl, err := g.Advertise(ctx, testNs, discovery.TTL(advertiseTTL))
// 	require.NoError(t, err, "should advertise successfully")
// 	require.Equal(t, advertiseTTL, ttl, "should return advertised TTL")

// 	// FindPeers ...
// 	finder, err := g.FindPeers(ctx, testNs)
// 	require.NoError(t, err, "should issue request packet")

// 	select {
// 	case <-time.After(time.Second):
// 		t.Error("should receive response")
// 	case info := <-finder:
// 		// NOTE: we advertised h's record to avoid creating a separate host
// 		require.Equal(t, info, *host.InfoFromHost(h))
// 	}
// }
