package ctxutil

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pkg/errors"
)

// WithLifetime returns a context that expires when the process receives any of the
// following signals from the operating system:
// - SIGINT
// - SIGTERM
// - SIGKILL
func WithLifetime(ctx context.Context) context.Context {
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
