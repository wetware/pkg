package cluster

import (
	"crypto/rand"
	"testing"
	"time"

	"capnproto.org/go/capnp/v3"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
	chan_api "github.com/wetware/ww/internal/api/channel"
)

func TestParam(t *testing.T) {
	t.Parallel()

	const size = 8

	_, s := capnp.NewSingleSegmentMessage(nil)
	p, err := chan_api.NewRootSender_send_Params(s)
	require.NoError(t, err)

	ps := sendParams(p)

	t.Run("CreateAndReadRecords", func(t *testing.T) {
		want, err := ps.NewRecords(size)
		require.NoError(t, err)
		require.Equal(t, size, want.Len())

		for i := 0; i < want.Len(); i++ {
			r := want.At(i)
			err = r.Bind(record{
				id:  newID(),
				ttl: time.Duration(i),
				seq: uint64(i),
			})
			require.NoError(t, err, "bind record %d", i)
		}

		got, err := ps.Records()
		require.NoError(t, err)
		require.Equal(t, size, got.Len())

		for i := 0; i < got.Len(); i++ {
			require.Equal(t, time.Duration(i), got.At(i).TTL())
			require.Equal(t, uint64(i), got.At(i).Seq())
			require.Equal(t, want.At(i).Peer(), got.At(i).Peer())
		}
	})

	t.Run("MarshalAndUnmarshal", func(t *testing.T) {
		b, err := ps.Message().MarshalPacked()
		require.NoError(t, err)
		require.NotNil(t, b)

		m, err := capnp.UnmarshalPacked(b)
		require.NoError(t, err)
		require.NotNil(t, m)

		p, err := chan_api.ReadRootSender_send_Params(m)
		require.NoError(t, err)

		ptr, err := p.Value()
		require.NoError(t, err)
		require.True(t, ptr.IsValid())

		ps2 := sendParams(p)

		want, err := ps.Records()
		require.NoError(t, err)
		require.Equal(t, size, want.Len())

		got, err := ps2.Records()
		require.NoError(t, err)
		require.Equal(t, size, got.Len())

		for i := 0; i < got.Len(); i++ {
			require.Equal(t, time.Duration(i), got.At(i).TTL())
			require.Equal(t, uint64(i), got.At(i).Seq())
			require.Equal(t, want.At(i).Peer(), got.At(i).Peer())
		}
	})
}

func newID() peer.ID {
	pk, _, err := crypto.GenerateECDSAKeyPair(rand.Reader)
	if err != nil {
		panic(err)
	}

	id, err := peer.IDFromPrivateKey(pk)
	if err != nil {
		panic(err)
	}

	return id
}

type record struct {
	id  peer.ID
	ttl time.Duration
	seq uint64
}

func (r record) Peer() peer.ID      { return peer.ID(r.id) }
func (r record) TTL() time.Duration { return r.ttl }
func (r record) Seq() uint64        { return r.seq }
