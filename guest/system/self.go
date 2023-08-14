package system

import (
	"context"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/pkg/api/process"
	"github.com/wetware/pkg/cap/csp"
)

type Self struct {
	Args []string
	Caps []capnp.Client
	CID  string
	PID  uint32
}

func Init(ctx context.Context) (Self, error) {
	var (
		s   Self
		err error
	)

	b := Bootstrap[api.BootContext](ctx)
	bCtx := csp.BootCtx(b)

	s.Args, err = bCtx.Args(ctx)
	if err != nil {
		return s, err
	}

	s.Caps, err = bCtx.Caps(ctx)
	if err != nil {
		return s, err
	}

	s.CID, err = bCtx.Cid(ctx)
	if err != nil {
		return s, err
	}

	s.PID, err = bCtx.Pid(ctx)
	return s, err
}
