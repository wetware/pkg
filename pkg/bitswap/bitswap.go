//go:generate mockgen -source=bitswap.go -destination=../../internal/mock/pkg/bitswap/bitswap.go -package=mock_bitswap

package bitswap

import (
	"context"
	"errors"

	"capnproto.org/go/capnp/v3"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	api "github.com/wetware/ww/internal/api/bitswap"
)

// Exchange is the local peer's BitSwap exchange.  It is wrapped by Server
// to provide the BitSwap capability over the network.
type Exchange interface {
	GetBlock(context.Context, cid.Cid) (blocks.Block, error)
}

type BitSwap api.BitSwap

func (bs BitSwap) AddRef() BitSwap {
	return BitSwap(api.BitSwap(bs).AddRef())
}

func (bs BitSwap) Release() {
	api.BitSwap(bs).Release()
}

// GetBlocks attempts to resolve the block corresponding to the supplied CID.
// It relies on ctx to time-out or cancel.  The block received by the client
// is transparently verified against the supplied key, and blocks.ErrWrongHash
// is returned if these do not match.
func (bs BitSwap) GetBlock(ctx context.Context, key cid.Cid) (blocks.Block, error) {
	if !key.Defined() {
		return nil, cid.ErrInvalidCid{Err: errors.New("null key")}
	}

	f, release := api.BitSwap(bs).GetBlock(ctx, func(call api.BitSwap_getBlock_Params) error {
		return call.SetKey(key.Bytes())
	})
	defer release()

	res, err := f.Struct()
	if err != nil {
		return nil, err
	}

	data, err := res.Block()
	if err != nil {
		return nil, err
	}

	if b := blocks.NewBlock(data); b.Cid().Equals(key) {
		return b, nil
	}

	return nil, blocks.ErrWrongHash
}

type Server struct {
	Exchange Exchange
}

func (s Server) BitSwap() BitSwap {
	return BitSwap(api.BitSwap_ServerToClient(s))
}

func (s Server) Client() capnp.Client {
	return capnp.Client(s.BitSwap())
}

func (s Server) GetBlock(ctx context.Context, call api.BitSwap_getBlock) error {
	b, err := call.Args().Key()
	if err != nil {
		return err
	}

	key, err := cid.Cast(b)
	if err != nil {
		return err
	}

	call.Go()

	block, err := s.Exchange.GetBlock(ctx, key)
	if err != nil {
		return err
	}

	res, err := call.AllocResults()
	if err == nil {
		err = res.SetBlock(block.RawData())
	}

	return err
}
