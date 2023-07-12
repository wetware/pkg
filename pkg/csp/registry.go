package csp

import (
	"context"

	api "github.com/wetware/ww/api/process"
)

type Registry api.BytecodeRegistry

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
	return res.Md5sum()
}

func (r Registry) Get(ctx context.Context, md5sum []byte) ([]byte, error) {
	f, release := api.BytecodeRegistry(r).Get(ctx, func(br api.BytecodeRegistry_get_Params) error {
		return br.SetMd5sum(md5sum)
	})
	defer release()

	<-f.Done()
	res, err := f.Struct()
	if err != nil {
		return nil, err
	}
	return res.Bytecode()
}

func (r Registry) Has(ctx context.Context, md5sum []byte) (bool, error) {
	f, release := api.BytecodeRegistry(r).Get(ctx, func(br api.BytecodeRegistry_get_Params) error {
		return br.SetMd5sum(md5sum)
	})
	defer release()

	<-f.Done()
	res, err := f.Struct()
	if err != nil {
		return false, err
	}
	return res.HasBytecode(), nil
}
