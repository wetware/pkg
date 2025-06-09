package csp_server

import (
	"context"

	capnp "capnproto.org/go/capnp/v3"
	api "github.com/wetware/pkg/api/process"
)

type ProcessBootstrap struct {
	caps []capnp.Client
}

func NewProcessBootstrap() *ProcessBootstrap {
	return &ProcessBootstrap{
		caps: make([]capnp.Client, 0),
	}
}

func (p *ProcessBootstrap) pop() (capnp.Client, bool) {
	if len(p.caps) == 0 {
		return capnp.Client{}, false
	}
	cap := p.caps[0]
	p.caps = p.caps[1:]
	return cap, true
}

func (b *ProcessBootstrap) Add(ctx context.Context, call api.Bootstrap_add) error {
	b.AddDirect(call.Args().Capability().AddRef())
	return nil
}

func (b *ProcessBootstrap) AddDirect(cap capnp.Client) {
	b.caps = append(b.caps, cap)
}

func (b *ProcessBootstrap) Get(ctx context.Context, call api.Bootstrap_get) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	if len(b.caps) == 0 {
		return nil
	}

	cap, ok := b.pop()
	if !ok {
		return nil
	}

	return res.SetCapability(cap)
}

func (b *ProcessBootstrap) ToCap() api.Bootstrap {
	return api.Bootstrap_ServerToClient(b)
}
