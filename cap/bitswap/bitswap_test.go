package bitswap_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"github.com/wetware/pkg/cap/bitswap"
	test_bitswap "github.com/wetware/pkg/cap/bitswap/test"
)

var (
	matchCtx = gomock.AssignableToTypeOf(context.Background())
	matchCID = gomock.AssignableToTypeOf(cid.Cid{})
)

func TestBitSwap(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("Succeed", func(t *testing.T) {
		t.Parallel()

		/*
			Test the "happy path".
		*/

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		want := blocks.NewBlock([]byte("hello, world!"))

		ex := test_bitswap.NewMockExchange(ctrl)
		ex.EXPECT().
			GetBlock(matchCtx, matchCID).
			Return(want, nil).
			Times(1)

		bs := bitswap.Server{Exchange: ex}.BitSwap()
		got, err := bs.GetBlock(context.Background(), want.Cid())
		require.NoError(t, err)
		require.True(t, got.Cid().Equals(want.Cid()),
			"CIDs should match")
	})

	t.Run("ErrWrongHash", func(t *testing.T) {
		t.Parallel()

		/*
			Test that we fail if the block returned by the client does not
			correspond to the CID we requested.
		*/

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		want := blocks.NewBlock([]byte("hello, world!"))
		got := blocks.NewBlock([]byte("something else"))

		ex := test_bitswap.NewMockExchange(ctrl)
		ex.EXPECT().
			GetBlock(matchCtx, matchCID).
			Return(got, nil).
			Times(1)

		bs := bitswap.Server{Exchange: ex}.BitSwap()

		_, err := bs.GetBlock(context.Background(), want.Cid())
		require.ErrorIs(t, err, blocks.ErrWrongHash,
			"should fail due to hash mismatch")
	})

	t.Run("ErrInvalidKey", func(t *testing.T) {
		t.Parallel()

		/*
			Test that we fail if the zero-value cid.Cid{} is supplied as the key.
			This test also checks that the failure happens before any RPC data
			is written to the wire.
		*/

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ex := test_bitswap.NewMockExchange(ctrl)
		bs := bitswap.Server{Exchange: ex}.BitSwap()

		got, err := bs.GetBlock(context.Background(), cid.Cid{}) // oops!
		require.ErrorIs(t, err, cid.ErrInvalidCid{})
		require.Nil(t, got, "should not return block")
	})
}
