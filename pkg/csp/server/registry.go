package server

import (
	"context"
	"crypto/md5"

	api "github.com/wetware/ww/api/process"
)

// TODO mikel
// Make registryServer keep a list of the bcs it has sorted by last usage
// Set and enforce a limited list size and a limited memory size
type RegistryServer map[[md5.Size]byte][]byte

func (r RegistryServer) put(bc []byte) []byte {
	md5sum := md5.Sum(bc)
	if _, found := r[md5sum]; !found {
		r[md5sum] = bc
	}
	return md5sum[:]
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

	return res.SetMd5sum(r.put(bc))
}

func (r RegistryServer) get(md5sum []byte) []byte {
	return r[[md5.Size]byte(md5sum)]
}

func (r RegistryServer) Get(ctx context.Context, call api.BytecodeRegistry_get) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	md5sum, err := call.Args().Md5sum()
	if err != nil {
		return err
	}

	return res.SetBytecode(r.get(md5sum))
}

func (r RegistryServer) has(md5sum []byte) bool {
	return r.get(md5sum) != nil
}

func (r RegistryServer) Has(ctx context.Context, call api.BytecodeRegistry_has) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	md5sum, err := call.Args().Md5sum()
	if err != nil {
		return err
	}

	res.SetHas(r.has(md5sum))
	return nil
}
