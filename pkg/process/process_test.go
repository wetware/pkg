package process_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wetware/ww/pkg/process"
)

var errTest = errors.New("test")

func TestProcess_wait(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("NilError", func(t *testing.T) {
		t.Parallel()

		p := process.New(func(context.Context) error {
			return nil
		})

		err := p.Wait(context.TODO())
		assert.NoError(t, err, "should succeed immediately")
	})

	t.Run("NonError", func(t *testing.T) {
		t.Parallel()

		p := process.New(func(context.Context) error {
			return errTest
		})

		err := p.Wait(context.TODO())
		assert.ErrorIs(t, err, errTest, "should return errTest")
	})

	t.Run("ContextErrorsReported", func(t *testing.T) {
		t.Parallel()

		/*
			Check that context errors are returned as expected.
		*/

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()

		cherr := make(chan error)
		defer close(cherr)

		p := process.New(func(context.Context) error {
			return <-cherr
		})

		err := p.Wait(ctx)
		assert.ErrorIs(t, err, context.DeadlineExceeded, "should report context error")
	})

	t.Run("ShutdownAfterRelease", func(t *testing.T) {
		t.Parallel()

		/*
			Check that releasing a process causes the context to expire.
		*/

		var (
			callCtx context.Context
			sync    = make(chan struct{})
		)
		p := process.New(func(ctx context.Context) error {
			callCtx = ctx
			close(sync)
			<-ctx.Done()
			return nil
		})

		<-sync

		p.AddRef().Release()
		assert.NoError(t, callCtx.Err(), "context should not expire")

		p.Release() // drop the only reference to p
		assert.Error(t, callCtx.Err(), "context should have expired")
	})
}
