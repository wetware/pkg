package csp

import (
	"context"
	"crypto/md5"

	api "github.com/wetware/pkg/api/process"
)

// HashSize is the size of the hash produced by HashFunc.
const HashSize = md5.Size

// HashFunc is the function used for hashing in the default
// executor implementation.
// TODO switch to more suitable hashing function, e.g. BLAKE3.
var HashFunc func([]byte) [HashSize]byte = md5.Sum

type Cache api.BytecodeCache

// Put a bytecode in the Cache with it's HashFunc as the key.
func (c Cache) Put(ctx context.Context, bc []byte) ([]byte, error) {
	f, release := api.BytecodeCache(c).Put(ctx, func(params api.BytecodeCache_put_Params) error {
		return params.SetBytecode(bc)
	})
	defer release()

	<-f.Done()
	res, err := f.Struct()
	if err != nil {
		return nil, err
	}
	return res.Hash()
}

// Get the bytecode associated to the hash produced by HashFunc(bytecode).
func (c Cache) Get(ctx context.Context, hash []byte) ([]byte, error) {
	f, release := api.BytecodeCache(c).Get(ctx, func(params api.BytecodeCache_get_Params) error {
		return params.SetHash(hash)
	})
	defer release()

	<-f.Done()
	res, err := f.Struct()
	if err != nil {
		return nil, err
	}
	return res.Bytecode()
}

// Has returns whether there is a match for the hash or not.
func (c Cache) Has(ctx context.Context, hash []byte) (bool, error) {
	f, release := api.BytecodeCache(c).Get(ctx, func(params api.BytecodeCache_get_Params) error {
		return params.SetHash(hash)
	})
	defer release()

	<-f.Done()
	res, err := f.Struct()
	if err != nil {
		return false, err
	}
	return res.HasBytecode(), nil
}
