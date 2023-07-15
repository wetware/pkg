package server

import (
	"context"

	api "github.com/wetware/ww/api/process"
	"github.com/wetware/ww/pkg/csp"
)

// TODO mikel
// Make registryServer keep a list of the bcs it has sorted by last usage
// Set and enforce a limited list size and a limited memory size
type RegistryServer map[[csp.HashSize]byte][]byte

func (r RegistryServer) put(bc []byte) []byte {
	hash := csp.HashFunc(bc)
	if _, found := r[hash]; !found {
		cached := make([]byte, len(bc))
		copy(cached, bc)
		r[hash] = cached
	}
	return hash[:]
}

func (r RegistryServer) Put(ctx context.Context, call api.BytecodeRegistry_put) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	bc, err := call.Args().Bytecode()
	if err != nil {
		return err
	}

	return res.SetHash(r.put(bc))
}

func (r RegistryServer) get(hash []byte) []byte {
	return r[[csp.HashSize]byte(hash)]
}

func (r RegistryServer) Get(ctx context.Context, call api.BytecodeRegistry_get) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	hash, err := call.Args().Hash()
	if err != nil {
		return err
	}

	return res.SetBytecode(r.get(hash))
}

func (r RegistryServer) has(hash []byte) bool {
	return r.get(hash) != nil
}

func (r RegistryServer) Has(ctx context.Context, call api.BytecodeRegistry_has) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	hash, err := call.Args().Hash()
	if err != nil {
		return err
	}

	res.SetHas(r.has(hash))
	return nil
}
