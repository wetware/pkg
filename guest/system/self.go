package system

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"github.com/ipfs/go-cid"
	api "github.com/wetware/pkg/api/process"
	"github.com/wetware/pkg/cap/csp"
)

type Self struct {
	Args    []string
	Caps    []capnp.Client
	CID     cid.Cid
	PID     uint32
	Boot    api.BootContext
	Release capnp.ReleaseFunc
}

func Init(ctx context.Context) (s Self, err error) {
	s.Boot, s.Release = Bootstrap[api.BootContext](ctx)

	s.Args, err = s.BootContext().Args(ctx)
	if err != nil {
		return s, err
	}

	s.Caps, err = s.BootContext().Caps(ctx)
	if err != nil {
		return s, err
	}

	s.CID, err = s.BootContext().Cid(ctx)
	if err != nil {
		return s, err
	}

	s.PID, err = s.BootContext().Pid(ctx)
	return s, err
}

func (s Self) BootContext() csp.BootCtx {
	return csp.BootCtx(s.Boot)
}
