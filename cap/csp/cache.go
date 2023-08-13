package csp

import (
	"context"

	api "github.com/wetware/pkg/api/process"
)

type Cache api.BytecodeCache

// Put a bytecode in the Cache with it's CidFunc as the key.
func (c Cache) Put(ctx context.Context, bc []byte) (string, error) {
	f, release := api.BytecodeCache(c).Put(ctx, func(params api.BytecodeCache_put_Params) error {
		return params.SetBytecode(bc)
	})
	defer release()

	<-f.Done()
	res, err := f.Struct()
	if err != nil {
		return "", err
	}
	return res.Cid()
}

// Get the bytecode associated to the cid produced by CidFunc(bytecode).
func (c Cache) Get(ctx context.Context, cid string) ([]byte, error) {
	f, release := api.BytecodeCache(c).Get(ctx, func(params api.BytecodeCache_get_Params) error {
		return params.SetCid(cid)
	})
	defer release()

	<-f.Done()
	res, err := f.Struct()
	if err != nil {
		return nil, err
	}
	return res.Bytecode()
}

// Has returns whether there is a match for the cid or not.
func (c Cache) Has(ctx context.Context, cid string) (bool, error) {
	f, release := api.BytecodeCache(c).Get(ctx, func(params api.BytecodeCache_get_Params) error {
		return params.SetCid(cid)
	})
	defer release()

	<-f.Done()
	res, err := f.Struct()
	if err != nil {
		return false, err
	}
	return res.HasBytecode(), nil
}
