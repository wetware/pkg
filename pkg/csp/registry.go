package csp

import (
	"context"
	"crypto/md5"

	api "github.com/wetware/ww/api/process"
)

// HashSize is the size of the hash produced by HashFunc.
const HashSize = md5.Size

// HashFunc is the function used for hashing in the default
// executor implementation.
// TODO switch to more suitable hashing function, e.g. BLAKE3.
var HashFunc func([]byte) [HashSize]byte = md5.Sum

type Registry api.BytecodeRegistry

// Put a bytecode in the registry with it's HashFunc as the key.
func (r Registry) Put(ctx context.Context, bc []byte) ([]byte, error) {
	f, release := api.BytecodeRegistry(r).Put(ctx, func(br api.BytecodeRegistry_put_Params) error {
		return br.SetBytecode(bc)
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
func (r Registry) Get(ctx context.Context, hash []byte) ([]byte, error) {
	f, release := api.BytecodeRegistry(r).Get(ctx, func(br api.BytecodeRegistry_get_Params) error {
		return br.SetHash(hash)
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
func (r Registry) Has(ctx context.Context, hash []byte) (bool, error) {
	f, release := api.BytecodeRegistry(r).Get(ctx, func(br api.BytecodeRegistry_get_Params) error {
		return br.SetHash(hash)
	})
	defer release()

	<-f.Done()
	res, err := f.Struct()
	if err != nil {
		return false, err
	}
	return res.HasBytecode(), nil
}
