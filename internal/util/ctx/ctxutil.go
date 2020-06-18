package ctxutil

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pkg/errors"
	"go.uber.org/fx"
)

// WithLifecycle returns a context that is bound to a go.uber.org/fx Lifecycle.
func WithLifecycle(ctx context.Context, lx fx.Lifecycle) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	lx.Append(fx.Hook{
		OnStop: func(context.Context) error {
			cancel()
			return nil
		},
	})

	return ctx
}

// WithDefaultSignals returns a context that expires when the process receives any of
// the following signals from the operating system:
// - SIGINT
// - SIGTERM
// - SIGKILL
func WithDefaultSignals(ctx context.Context) context.Context {
	return WithSignals(ctx, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
}

// WithSignals returns a context that expires when the process receives any of the
// specified signals.
func WithSignals(ctx context.Context, sigs ...os.Signal) context.Context {
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, sigs...)

	cq := make(chan struct{})
	sctx := &sigctx{
		cq:      cq,
		Context: ctx,
	}

	go func() {
		defer close(cq)

		select {
		case sig := <-sigch:
			sctx.mu.Lock()
			defer sctx.mu.Unlock()

			sctx.err = errors.Errorf("signal received: %s", sig)
		case <-ctx.Done():
			sctx.mu.Lock()
			defer sctx.mu.Unlock()

			sctx.err = ctx.Err()
		}
	}()

	return sctx
}

type sigctx struct {
	mu  sync.RWMutex
	err error

	cq <-chan struct{}
	context.Context
}

func (ctx *sigctx) Done() <-chan struct{} {
	return ctx.cq
}

func (ctx *sigctx) Err() (err error) {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	return ctx.err
}
