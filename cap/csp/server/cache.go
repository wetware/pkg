package csp_server

import (
	"context"

	api "github.com/wetware/pkg/api/process"
	"github.com/wetware/pkg/cap/csp"
)

// TODO mikel
// Make BytecodeCache keep a list of the bcs it has sorted by last usage
// Set and enforce a limited list size and a limited memory size
type BytecodeCache map[[csp.HashSize]byte][]byte

func (c BytecodeCache) put(bc []byte) []byte {
	hash := csp.HashFunc(bc)
	if _, found := c[hash]; !found {
		cached := make([]byte, len(bc))
		copy(cached, bc)
		c[hash] = cached
	}
	return hash[:]
}

func (c BytecodeCache) Put(ctx context.Context, call api.BytecodeCache_put) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	bc, err := call.Args().Bytecode()
	if err != nil {
		return err
	}

	return res.SetHash(c.put(bc))
}

func (c BytecodeCache) get(hash []byte) []byte {
	return c[[csp.HashSize]byte(hash)]
}

func (c BytecodeCache) Get(ctx context.Context, call api.BytecodeCache_get) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	hash, err := call.Args().Hash()
	if err != nil {
		return err
	}

	return res.SetBytecode(c.get(hash))
}

func (c BytecodeCache) has(hash []byte) bool {
	return c.get(hash) != nil
}

func (c BytecodeCache) Has(ctx context.Context, call api.BytecodeCache_has) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	hash, err := call.Args().Hash()
	if err != nil {
		return err
	}

	res.SetHas(c.has(hash))
	return nil
}
