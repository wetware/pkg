package proc

import (
	"context"
	"runtime"

	"github.com/wetware/ww/pkg/cap/proc/unix"
)

type UnixProc struct {
	Client unix.Client
}

func (p *UnixProc) UnixExec(ctx context.Context, name string, args ...string) *Process {
	capProc, release := p.Client.Exec(ctx, name, args...)
	proc := &Process{client: capProc}

	runtime.SetFinalizer(proc, release)

	return proc
}
