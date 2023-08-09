package socket_test

import (
	"testing"

	"capnproto.org/go/capnp/v3"
	"github.com/stretchr/testify/require"
	boot "github.com/wetware/ww/api/boot"
	"github.com/wetware/ww/boot/socket"
)

func TestRecord_Peer(t *testing.T) {
	t.Parallel()

	_, seg := capnp.NewSingleSegmentMessage(nil)
	packet, err := boot.NewRootPacket(seg)
	require.NoError(t, err)

	id := newPeerID()

	packet.SetRequest()
	err = packet.Request().SetFrom(string(id))
	require.NoError(t, err)

	rec := socket.Record(packet)
	got, err := rec.Peer()
	require.NoError(t, err)
	require.Equal(t, id, got)

}
