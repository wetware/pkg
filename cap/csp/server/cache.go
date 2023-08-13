package csp_server

import (
	"context"

	api "github.com/wetware/pkg/api/process"
	"github.com/wetware/pkg/rom"
)

// TODO mikel
// Make BytecodeCache keep a list of the bcs it has sorted by last usage
// Set and enforce a limited list size and a limited memory size
type BytecodeCache map[string][]byte

func (c BytecodeCache) put(bc []byte) string {
	rom := rom.ROM{Bytecode: bc}
	cid := rom.CID().String()
	if _, found := c[cid]; !found {
		cached := make([]byte, len(bc))
		copy(cached, bc)
		c[cid] = cached
	}
	return cid
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

	return res.SetCid(c.put(bc))
}

func (c BytecodeCache) get(cid string) []byte {
	return c[cid]
}

func (c BytecodeCache) Get(ctx context.Context, call api.BytecodeCache_get) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	cid, err := call.Args().Cid()
	if err != nil {
		return err
	}

	return res.SetBytecode(c.get(cid))
}

func (c BytecodeCache) has(cid string) bool {
	return c.get(cid) != nil
}

func (c BytecodeCache) Has(ctx context.Context, call api.BytecodeCache_has) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	cid, err := call.Args().Cid()
	if err != nil {
		return err
	}

	res.SetHas(c.has(cid))
	return nil
}
