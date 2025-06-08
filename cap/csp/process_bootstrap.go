package csp

import (
	"context"

	capnp "capnproto.org/go/capnp/v3"
	api "github.com/wetware/pkg/api/process"
)

type ProcessBootstrap api.Bootstrap

func (p ProcessBootstrap) Add(ctx context.Context, cap capnp.Client) error {
	f, release := api.Bootstrap(p).Add(ctx, func(ps api.Bootstrap_add_Params) error {
		return ps.SetCapability(cap)
	})
	defer release()

	select {
	case <-f.Done():
	case <-ctx.Done():
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	_, err := f.Struct()
	if err != nil {
		return err
	}
	return nil
}

func (p ProcessBootstrap) Get(ctx context.Context) (capnp.Client, capnp.ReleaseFunc, bool, error) {
	f, release := api.Bootstrap(p).Get(ctx, nil)

	select {
	case <-f.Done():
	case <-ctx.Done():
	}

	if ctx.Err() != nil {
		return capnp.Client{}, release, false, ctx.Err()
	}

	res, err := f.Struct()
	if err != nil {
		return capnp.Client{}, release, false, err
	}

	if !res.HasCapability() {
		return capnp.Client{}, release, false, nil
	}

	return f.Capability(), release, true, nil
}
