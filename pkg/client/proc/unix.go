package proc

import (
	"context"

	"github.com/wetware/ww/pkg/cap/proc/unix"
)

type UnixProc struct {
	Client unix.Client
}

func (p *UnixProc) Exec(ctx context.Context, name string, args ...string) *Process {
	return newProcess(p.Client.Exec(ctx, name, args...))
}
