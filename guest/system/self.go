package system

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"github.com/ipfs/go-cid"
)

type Self struct {
	Args []string
	Caps []capnp.Client
	CID  cid.Cid
	PID  uint32
}

func Init(ctx context.Context) (Self, error) {
	// var (
	// 	s   Self
	// 	err error
	// )

	// b, err := Bootstrap(ctx)
	// if err != nil {
	// 	return Self{}, err
	// }

	panic("NOT IMPLEMENTED")

	// bCtx := csp.BootCtx(b)

	// s.Args, err = bCtx.Args(ctx)
	// if err != nil {
	// 	return s, err
	// }

	// s.Caps, err = bCtx.Caps(ctx)
	// if err != nil {
	// 	return s, err
	// }

	// s.CID, err = bCtx.Cid(ctx)
	// if err != nil {
	// 	return s, err
	// }

	// s.PID, err = bCtx.Pid(ctx)
	// return s, err
}
