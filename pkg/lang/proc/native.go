package proc

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/jbenet/goprocess"
	"github.com/spy16/parens"
	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/mem"
	capnp "zombiezen.com/go/capnproto2"
	"zombiezen.com/go/capnproto2/server"
)

var (
	_ Proc = (*goroutine)(nil)
)

func init() { Register("go", goroutineFactory) }

func goroutineFactory(env *parens.Env, args []ww.Any) (Proc, error) {
	var err error
	var any parens.Any

	target := args[0]
	g := &goroutine{
		args: args,
		proc: goprocess.Go(func(p goprocess.Process) {
			any, err = env.Eval(target)
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

	if err := val.Raw.SetProc(api.Proc_ServerToClient(procCap{g}, &server.Policy{})); err != nil {
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
