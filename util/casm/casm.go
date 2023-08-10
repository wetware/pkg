package casm

import (
	"context"

	"capnproto.org/go/capnp/v3"
)

type Future struct{ *capnp.Future }

func (f Future) Err() error {
	_, err := f.Struct()
	return err
}

func (f Future) Await(ctx context.Context) error {
	select {
	case <-f.Done():
	case <-ctx.Done():
	}

	// The future may have resolved due to a canceled context, in which
	// case there is a race-condition in the above select.
	if ctx.Err() != nil {
		return ctx.Err()
	}

	return f.Err()
}

type Iterator[T any] struct {
	Seq interface {
		Next() (T, bool)
	}

	Future interface {
		Done() <-chan struct{}
		Err() error
	}
}

// Err returns returns the first error encountered while iterating
// over the stream.   Callers SHOULD call Err() after the iterator
// has become exhausted, and handle any errors.
func (it Iterator[T]) Err() (err error) {
	if it.Future != nil {
		select {
		case <-it.Future.Done():
			err = it.Future.Err()
		default:
		}
	}

	return
}

func (it Iterator[T]) Next() (t T, ok bool) {
	if it.Seq != nil {
		t, ok = it.Seq.Next()
	}

	return
}
