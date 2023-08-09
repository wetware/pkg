package pulse_test

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/cluster/pulse"
	mock_pulse "github.com/wetware/ww/internal/mock/cluster/pulse"
)

var reader = rand.New(rand.NewSource(42))

func TestValidator(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("Accept_upserted_record", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		hb := pulse.NewHeartbeat()
		hb.SetTTL(time.Hour)

		b, err := hb.Message().MarshalPacked()
		require.NoError(t, err)

		id := newPeerID()
		msg := &pubsub.Message{Message: &pb.Message{
			From:  []byte(id),
			Seqno: []byte{0, 0, 0, 0, 0, 0, 0, 1}, // Seq=1
			Data:  b,
		}}

		rt := mock_pulse.NewMockRoutingTable(ctrl)
		rt.EXPECT().
			Upsert(gomock.Any()).
			Return(true).
			Times(1)

		validate := pulse.NewValidator(rt)

		res := validate(context.Background(), id, msg)
		require.Equal(t, pubsub.ValidationAccept, res)

	})

	t.Run("Reject_malformed_record", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		rt := mock_pulse.NewMockRoutingTable(ctrl)
		validate := pulse.NewValidator(rt)

		msg := &pubsub.Message{Message: &pb.Message{}}
		res := validate(context.Background(), newPeerID(), msg)
		assert.Equal(t, pubsub.ValidationReject, res)
	})

	t.Run("Ignore_stale_record", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		hb := pulse.NewHeartbeat()
		hb.SetTTL(time.Hour)

		b, err := hb.Message().MarshalPacked()
		require.NoError(t, err)

		id := newPeerID()
		first := &pubsub.Message{Message: &pb.Message{
			From:  []byte(id),
			Seqno: []byte{0, 0, 0, 0, 0, 0, 0, 8}, // Seq=8
			Data:  b,
		}}

		second := &pubsub.Message{Message: &pb.Message{
			From:  []byte(id),
			Seqno: []byte{0, 0, 0, 0, 0, 0, 0, 1}, // Seq=1
			Data:  b,
		}}

		rt := mock_pulse.NewMockRoutingTable(ctrl)
		rt.EXPECT().
			Upsert(gomock.Any()).
			Return(true).
			Times(1)
		rt.EXPECT().
			Upsert(gomock.Any()).
			Return(false).
			Times(1)

		validate := pulse.NewValidator(rt)

		res := validate(context.Background(), id, first)
		require.Equal(t, pubsub.ValidationAccept, res)

		res = validate(context.Background(), id, second)
		require.Equal(t, pubsub.ValidationIgnore, res)
	})
}

func newPeerID() peer.ID {
	sk, _, err := crypto.GenerateEd25519Key(reader)
	if err != nil {
		panic(err)
	}

	id, err := peer.IDFromPrivateKey(sk)
	if err != nil {
		panic(err)
	}

	return id
}
