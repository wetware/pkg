package proc

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/jbenet/goprocess"
	"github.com/spy16/parens"
	capnp "zombiezen.com/go/capnproto2"
	"zombiezen.com/go/capnproto2/server"

	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/core"
	"github.com/wetware/ww/pkg/mem"
)

var (
	_ core.Process = (*goroutine)(nil)
)

func init() { core.RegisterProcessFactory("go", goroutineFactory) }

func goroutineFactory(env core.Env, args []ww.Any) (core.Process, error) {
	var err error
	var any parens.Any

	// target := args[0]
	g := &goroutine{
		args: args,
		proc: goprocess.Go(func(p goprocess.Process) {
			panic("FIXME")
			// any, err = env.Eval(target)
		}),
	}

	g.proc.SetTeardown(func() error {
		g.res.Store(any.(ww.Any))
		return err
	})

	return g, nil
}

type goroutine struct {
	res  atomic.Value
	proc goprocess.Process
	args []ww.Any
}

func (g goroutine) String() string {
	select {
	case <-g.proc.Closed():
		if err := g.proc.Err(); err != nil {
			return fmt.Sprintf("<Goroutine [ERR: %s]>", err)
		}

		return fmt.Sprintf("<Goroutine [%#v]>", g.res.Load())

	default:
		return "<Goroutine [running]>"
	}
}

func (g goroutine) MemVal() mem.Value {
	val, err := mem.NewValue(capnp.SingleSegment(nil))
	if err != nil {
		panic(err)
	}

	if err := val.Raw.SetProc(api.Proc_ServerToClient(gCap{g}, &server.Policy{})); err != nil {
		panic(err)
	}

	return val
}

func (g goroutine) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-g.proc.Closed():
		return g.proc.Err()
	}
}

func (g goroutine) Result() ww.Any { return g.res.Load().(ww.Any) }

type gCap struct{ core.Process }

func (p gCap) Wait(ctx context.Context, call api.Proc_wait) error {
	return p.Process.Wait(ctx)
}
