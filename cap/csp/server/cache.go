package csp_server

import (
	"context"

	"github.com/ipfs/go-cid"
	api "github.com/wetware/pkg/api/process"
	"github.com/wetware/pkg/rom"
)

// TODO mikel
// Make BytecodeCache keep a list of the bcs it has sorted by last usage
// Set and enforce a limited list size and a limited memory size
type BytecodeCache map[string][]byte

func (c BytecodeCache) put(bc []byte) cid.Cid {
	rom := rom.ROM{Bytecode: bc}
	cid := rom.CID()
	key := cid.String()
	if _, found := c[key]; !found {
		cached := make([]byte, len(bc))
		copy(cached, bc)
		c[key] = cached
	}
	return cid
}

func (c BytecodeCache) ExposedPut(bc []byte) cid.Cid {
	return c.put(bc)
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

	cid := c.put(bc)
	return res.SetCid(cid.Bytes())
}

func (c BytecodeCache) get(cid cid.Cid) []byte {
	return c[cid.String()]
}

func (c BytecodeCache) Get(ctx context.Context, call api.BytecodeCache_get) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	b, err := call.Args().Cid()
	if err != nil {
		return err
	}

	_, cid, err := cid.CidFromBytes(b)
	if err != nil {
		return err
	}

	return res.SetBytecode(c.get(cid))
}

func (c BytecodeCache) has(cid cid.Cid) bool {
	if cid.ByteLen() == 0 {
		return false
	}

	return c.get(cid) != nil
}

func (c BytecodeCache) Has(ctx context.Context, call api.BytecodeCache_has) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	b, err := call.Args().Cid()
	if err != nil {
		return err
	}

	_, cid, err := cid.CidFromBytes(b)
	if err != nil {
		return err
	}

	res.SetHas(c.has(cid))
	return nil
}
