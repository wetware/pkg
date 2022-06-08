package process_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wetware/ww/pkg/ocap/process"
)

var errTest = errors.New("test")

func TestProcess_wait(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("NilError", func(t *testing.T) {
		t.Parallel()

		p := process.New(context.TODO(), func(context.Context) error {
			return nil
		})

		err := p.Wait(context.TODO())
		assert.NoError(t, err, "should succeed immediately")
	})

	t.Run("NonError", func(t *testing.T) {
		t.Parallel()

		p := process.New(context.TODO(), func(context.Context) error {
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

		p := process.New(context.TODO(), func(context.Context) error {
			return <-cherr
		})

		err := p.Wait(ctx)
		assert.ErrorIs(t, err, context.DeadlineExceeded, "should report context error")
	})

	t.Run("KillWhenReleased", func(t *testing.T) {
		t.Parallel()

		p := process.New(context.TODO(), func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})

		// Not needed, but gets us 100% test coverage.  Forgive me...
		p.AddRef().Release()

		cherr := make(chan error, 1)
		go func() {
			cherr <- p.Wait(context.TODO())
		}()

		time.Sleep(time.Millisecond)
		p.Release()

		assert.ErrorIs(t, <-cherr, context.Canceled, "should report context error")
	})
}
